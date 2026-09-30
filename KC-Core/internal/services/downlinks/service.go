package downlinks

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Deps wires the Service; every entry is required.
type Deps struct {
	Queuer    Queuer
	Editor    PendingEditor
	Revoker   Revoker
	Endpoints EndpointLookup
	Events    UpdateRecorder
	Audit     AuditRecorder
	Log       logger.Logger
}

// Service queues, edits and revokes downlinks for an organization.
type Service struct {
	queuer    Queuer
	editor    PendingEditor
	revoker   Revoker
	endpoints EndpointLookup
	events    UpdateRecorder
	audit     AuditRecorder
	log       logger.Logger
}

// NewService validates the dependencies and builds the service.
func NewService(d Deps) (*Service, error) {
	if d.Queuer == nil || d.Editor == nil || d.Revoker == nil || d.Endpoints == nil || d.Events == nil || d.Audit == nil || d.Log == nil {
		return nil, ErrMissingDependency
	}
	return &Service{queuer: d.Queuer, editor: d.Editor, revoker: d.Revoker, endpoints: d.Endpoints, events: d.Events, audit: d.Audit, log: d.Log}, nil
}

// Queue checks the content and queues it for the owner's endpoint through
// the SCACI handler core; a failure there carries the SCACI catalog error.
func (s *Service) Queue(ctx context.Context, owner Owner, content Content) (*scaci.DLDataQueueResult, error) {
	checked, err := s.check(ctx, content)
	if err != nil {
		return nil, err
	}
	orgID := owner.OrganizationID
	result, err := s.queuer.QueueDownlinkInternal(ctx, owner.TenantID, &orgID, checked.queueRequest(owner.EpEUI), "")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrQueue, err)
	}
	s.log.InfoContext(ctx, LogDownlinkQueued,
		logger.FieldQueueID, result.QueID,
		logger.FieldBsEui, mioty.FormatEUI64(result.BsEui),
		logger.FieldOpID, result.OpID,
		logger.FieldTenantIDCamel, owner.TenantID,
		logger.FieldEpEui, mioty.FormatEUI64(owner.EpEUI))
	s.recordQueued(ctx, owner, result)
	return result, nil
}

// Update checks the content and rewrites the pending downlink with it;
// storage.ErrDownlinkNotFound when the owner has no such downlink,
// storage.ErrDownlinkNotPending when a base station holds it already.
func (s *Service) Update(ctx context.Context, target Target, content Content) (*storage.DownlinkMessage, error) {
	checked, err := s.check(ctx, content)
	if err != nil {
		return nil, err
	}
	orgID := target.OrganizationID
	updated, err := s.editor.UpdatePendingDownlink(ctx, target.TenantID, &orgID, mioty.EUI64Bytes(target.EpEUI), target.QueID, checked.patch())
	if errors.Is(err, storage.ErrDownlinkNotFound) || errors.Is(err, storage.ErrDownlinkNotPending) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpdate, err)
	}
	if err := s.events.RecordPendingUpdated(ctx, updated); err != nil {
		s.log.ErrorContext(ctx, LogDownlinkUpdateNotAnnounced, logger.FieldQueueID, target.QueID, logger.FieldError, err)
	}
	epEUI := mioty.FormatEUI64(target.EpEUI)
	s.audit.Record(ctx, audit.Event{
		TenantID:    target.TenantID,
		EventType:   models.EventTypeDownlinkUpdated,
		Title:       models.EventTitleDownlinkUpdated,
		Description: fmt.Sprintf(models.EventDescriptionDownlinkUpdated, target.QueID, epEUI),
		Details:     map[string]any{bssci.EventKeyEpEui: epEUI, bssci.EventKeyQueID: target.QueID},
	})
	return updated, nil
}

// Revoke revokes the owner's downlink where it waits. A queue id of another
// organization or endpoint reads like a missing one: the revoke path answers
// scheduler.ErrSchedulerQueueNotFound for it, wrapped in ErrRevoke.
func (s *Service) Revoke(ctx context.Context, target Target) (RevokeResult, error) {
	if err := s.requireEndpoint(ctx, target.Owner); err != nil {
		return RevokeResult{}, err
	}
	if target.QueID < 0 {
		return RevokeResult{}, fmt.Errorf("%w: %w", ErrRevoke, scheduler.ErrSchedulerQueueNotFound)
	}
	orgID, epEUI := target.OrganizationID, target.EpEUI
	station, err := s.revoker.RevokeDownlink(ctx, scheduler.DownlinkRef{
		TenantID: target.TenantID, QueID: uint64(target.QueID), OrganizationID: &orgID, EpEUI: &epEUI,
	})
	if err != nil {
		return RevokeResult{}, fmt.Errorf("%w: %w", ErrRevoke, err)
	}
	result := revokeResult(station)
	s.recordRevoke(ctx, target, result)
	return result, nil
}

// check validates the content, logging why it was refused.
func (s *Service) check(ctx context.Context, content Content) (checkedContent, error) {
	checked, err := content.check()
	if err != nil {
		s.log.WarnContext(ctx, LogDownlinkContentRefused, logger.FieldError, err)
	}
	return checked, err
}

// requireEndpoint refuses an endpoint the tenant has not registered.
func (s *Service) requireEndpoint(ctx context.Context, owner Owner) error {
	_, err := s.endpoints.GetByEUI(ctx, mioty.EUI64Bytes(owner.EpEUI), owner.TenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrEndpointNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEndpointLookup, err)
	}
	return nil
}
