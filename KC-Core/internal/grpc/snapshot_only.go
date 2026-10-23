package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SnapshotOnlyModeLookup reads whether a tenant decodes its endpoints only
// from their pinned blueprint snapshots.
type SnapshotOnlyModeLookup interface {
	GetSnapshotOnlyDecoding(ctx context.Context, tenantID int64) (bool, error)
}

// SnapshotOnlyModeStore reads and switches a tenant's snapshot-only decoding.
type SnapshotOnlyModeStore interface {
	SnapshotOnlyModeLookup
	SetSnapshotOnlyDecoding(ctx context.Context, tenantID int64, enabled bool) error
}

// SnapshotOnlyHandlers serves the tenant's snapshot-only decoding switch.
type SnapshotOnlyHandlers struct {
	modes SnapshotOnlyModeStore
	audit AuditRecorder
	log   logger.Logger
}

// SnapshotOnlyHandlerDeps wires SnapshotOnlyHandlers; a nil Modes store
// answers its RPCs as not configured.
type SnapshotOnlyHandlerDeps struct {
	Modes SnapshotOnlyModeStore
}

// NewSnapshotOnlyHandlers builds the group; recorder records every switch.
func NewSnapshotOnlyHandlers(d SnapshotOnlyHandlerDeps, recorder AuditRecorder, log logger.Logger) *SnapshotOnlyHandlers {
	return &SnapshotOnlyHandlers{modes: d.Modes, audit: recorder, log: log}
}

// Event description words for the switch's new value.
const (
	snapshotOnlyOn  = "on"
	snapshotOnlyOff = "off"
)

// SetSnapshotOnlyDecoding switches the caller's tenant.
func (s *SnapshotOnlyHandlers) SetSnapshotOnlyDecoding(ctx context.Context, req *pb.SetSnapshotOnlyDecodingRequest) (*pb.SnapshotOnlyDecoding, error) {
	if s.modes == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	enabled := req.GetEnabled()
	if err := s.modes.SetSnapshotOnlyDecoding(ctx, tenantID, enabled); err != nil {
		s.log.ErrorContext(ctx, LogSnapshotOnlyDecodingSetFailed, logger.FieldTenantIDSnake, tenantID, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSnapshotOnlySetFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSnapshotOnlySetFailed))
	}
	word := snapshotOnlyOff
	if enabled {
		word = snapshotOnlyOn
	}
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryEndpoint,
		EventType:   models.EventTypeSnapshotOnlyDecodingChanged,
		Title:       models.EventTitleSnapshotOnlyDecodingChanged,
		Description: fmt.Sprintf(models.EventDescriptionSnapshotOnlyDecodingChangedFmt, word),
		Details:     map[string]any{models.EventDetailKeySnapshotOnlyDecoding: enabled},
	})
	return &pb.SnapshotOnlyDecoding{Enabled: enabled}, nil
}

// GetSnapshotOnlyDecoding reads the caller's tenant.
func (s *SnapshotOnlyHandlers) GetSnapshotOnlyDecoding(ctx context.Context, _ *emptypb.Empty) (*pb.SnapshotOnlyDecoding, error) {
	if s.modes == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	enabled, err := s.modes.GetSnapshotOnlyDecoding(ctx, tenantID)
	if err != nil {
		return nil, snapshotOnlyReadError(ctx, s.log, tenantID, err)
	}
	return &pb.SnapshotOnlyDecoding{Enabled: enabled}, nil
}

// snapshotOnlyReadError logs a failed read of a tenant's switch and refuses the call.
func snapshotOnlyReadError(ctx context.Context, log logger.Logger, tenantID int64, err error) error {
	log.ErrorContext(ctx, LogSnapshotOnlyDecodingReadFailed, logger.FieldTenantIDSnake, tenantID, logger.FieldError, err)
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenSnapshotOnlyReadFailed),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenSnapshotOnlyReadFailed))
}

// snapshotOnly reads the switch of an endpoint's owner from storage on every
// call; a group built without the lookup decodes as before.
func snapshotOnly(ctx context.Context, modes SnapshotOnlyModeLookup, log logger.Logger, ownerTenantID int64) (bool, error) {
	if modes == nil {
		return false, nil
	}
	on, err := modes.GetSnapshotOnlyDecoding(ctx, ownerTenantID)
	if err != nil {
		return false, snapshotOnlyReadError(ctx, log, ownerTenantID, err)
	}
	return on, nil
}
