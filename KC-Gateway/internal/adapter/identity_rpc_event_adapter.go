// Package adapter bridges KC-Identity RPC responses to interceptor interfaces.
package adapter

import (
	"context"
	"strconv"
	"time"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"google.golang.org/grpc/metadata"
)

// IdentityRPCEventAdapter implements grpc.EventWriter by forwarding events
// to KC-Identity via the RecordPlatformEvent RPC. Used by KC-Gateway for
// lifecycle and security events since Gateway has no direct DB access.
type IdentityRPCEventAdapter struct {
	client           pb.IdentityInternalServiceClient
	peerSecret       string
	platformTenantID int64
	log              logger.Logger
}

// NewIdentityRPCEventAdapter creates an event adapter that persists events via KC-Identity.
func NewIdentityRPCEventAdapter(client pb.IdentityInternalServiceClient, peerSecret string, platformTenantID int64) *IdentityRPCEventAdapter {
	return &IdentityRPCEventAdapter{
		client:           client,
		peerSecret:       peerSecret,
		platformTenantID: platformTenantID,
		log:              logger.Get().WithField("component", "event-adapter"),
	}
}

// CreateEvent implements grpc.EventWriter by calling RecordPlatformEvent on KC-Identity.
func (a *IdentityRPCEventAdapter) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	if a.client == nil {
		return nil
	}

	// Parse tenant ID from string
	tenantID := a.platformTenantID
	if event.TenantID != "" {
		if tid, err := strconv.ParseInt(event.TenantID, 10, 64); err == nil {
			tenantID = tid
		}
	}

	// Marshal details to bytes
	var data []byte
	if event.Details != nil {
		data = event.Details
	}

	req := &pb.RecordPlatformEventRequest{
		TenantId:    tenantID,
		EventType:   event.EventType,
		Category:    event.Category,
		Severity:    event.Severity,
		SourceType:  event.SourceType,
		SourceName:  event.SourceName,
		UserId:      event.UserID,
		Title:       event.Title,
		Description: event.Description,
		Data:        data,
	}

	// Add peer auth metadata
	ctx = a.withPeerAuth(ctx)

	// Use a short timeout for event recording — don't block request processing
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := a.client.RecordPlatformEvent(rpcCtx, req); err != nil {
		a.log.Warn(LogGatewayPlatformEventViaIdentityRecordFailed,
			"eventType", event.EventType,
			"error", err)
		return err
	}

	return nil
}

// withPeerAuth adds the internal peer authentication metadata.
func (a *IdentityRPCEventAdapter) withPeerAuth(ctx context.Context) context.Context {
	if a.peerSecret == "" {
		return ctx
	}
	md := metadata.Pairs(grpcconstants.MetadataKeyInternalPeerSecret, a.peerSecret)
	return metadata.NewOutgoingContext(ctx, md)
}

// Verify interface compliance at compile time.
var _ grpcconstants.EventWriter = (*IdentityRPCEventAdapter)(nil)
