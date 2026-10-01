// Package bssciservices provides BSSCI-related services for KC-Core.
package bssciservices

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// AttachedEndpointCreator stores a new endpoint together with its attachment
// status, in one transaction.
type AttachedEndpointCreator interface {
	CreateWithStatus(ctx context.Context, endpoint *models.EndPoint, status string) (*models.EndPoint, error)
}

// endpointAttachmentService attaches and detaches a tenant's endpoints for
// the gRPC API: the decider records and announces each decision, and the
// connected base stations are sent it in the background.
type endpointAttachmentService struct {
	endpoints   EndpointAttachmentStore
	creator     AttachedEndpointCreator
	decider     bssci.AttachmentDecider
	notifier    EndpointStatusNotifier
	propagation *AttachmentPropagation
}

// NewEndpointAttachmentService creates the endpoint attachment service.
func NewEndpointAttachmentService(
	endpoints EndpointAttachmentStore,
	creator AttachedEndpointCreator,
	decider bssci.AttachmentDecider,
	notifier EndpointStatusNotifier,
	propagation *AttachmentPropagation,
) (grpcservices.EndpointAttachmentService, error) {
	switch {
	case endpoints == nil:
		return nil, errNilAttachmentEndpoints
	case creator == nil:
		return nil, errNilAttachedEndpointCreator
	case decider == nil:
		return nil, errNilAttachmentDecider
	case notifier == nil:
		return nil, errNilEndpointStatusNotifier
	case propagation == nil:
		return nil, errNilAttachmentPropagation
	}
	return &endpointAttachmentService{
		endpoints:   endpoints,
		creator:     creator,
		decider:     decider,
		notifier:    notifier,
		propagation: propagation,
	}, nil
}

// AttachEndPoint attaches the endpoint when it is called, announcing a change
// once, and propagates it to the connected base stations; one that connects
// later is sent it on connect (BSSCI §3.8).
func (s *endpointAttachmentService) AttachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	ep, err := s.tenantEndpoint(ctx, epEui, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errAttachEndPoint, err)
	}
	if err := requireAttachableKey(ep.NwkSnKey); err != nil {
		return nil, fmt.Errorf("%w: %w", errAttachEndPoint, err)
	}
	if _, err := s.decider.Decide(ctx, decisionFor(ep, tenantID, bssci.EndpointStatusAttached)); err != nil {
		return nil, fmt.Errorf("%w: %w", errAttachEndPoint, err)
	}
	return s.propagation.PropagateAttach(ctx, tenantID, ep), nil
}

// DetachEndPoint detaches the endpoint when it is called, announcing a change
// once, and propagates the detachment to the connected base stations; no
// station that connects later is sent it again.
func (s *endpointAttachmentService) DetachEndPoint(ctx context.Context, epEui string, tenantID int64) (*grpcservices.EndpointOperationResult, error) {
	ep, err := s.tenantEndpoint(ctx, epEui, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDetachEndPoint, err)
	}
	if _, err := s.decider.Decide(ctx, decisionFor(ep, tenantID, endpoint.EndpointStatusDetached)); err != nil {
		return nil, fmt.Errorf("%w: %w", errDetachEndPoint, err)
	}
	return s.propagation.PropagateDetach(ctx, tenantID, ep), nil
}

// CreateAttached stores a pre-attached endpoint attached (BSSCI §3.8): the
// endpoint and its attachment are stored together or not at all, and only
// then is the owner told and the connected base stations sent it.
func (s *endpointAttachmentService) CreateAttached(ctx context.Context, ep *models.EndPoint) (*models.EndPoint, error) {
	if err := requireAttachableKey(ep.NwkSnKey); err != nil {
		return nil, fmt.Errorf("%w: %w", errCreateAttachedEndpoint, err)
	}
	created, err := s.creator.CreateWithStatus(ctx, ep, bssci.EndpointStatusAttached)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errCreateAttachedEndpoint, err)
	}
	s.notifier.NotifyEndpointStatus(ctx, noticeOf(decisionFor(created, created.TenantID, bssci.EndpointStatusAttached)))
	s.propagation.PropagateAttach(ctx, created.TenantID, created)
	return created, nil
}

func (s *endpointAttachmentService) tenantEndpoint(ctx context.Context, epEui string, tenantID int64) (*models.EndPoint, error) {
	euiU64, err := validation.ParseEUI(epEui)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidEndpointEUI, err)
	}
	eui := models.EUI(mioty.EUI64(euiU64).ToBytes())
	ep, err := s.endpoints.GetByEUI(ctx, tenantID, eui[:])
	if errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("%w: %w", ErrEndpointNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGetEndpoint, err)
	}
	return ep, nil
}

func decisionFor(ep *models.EndPoint, tenantID int64, status string) bssci.AttachmentDecision {
	return bssci.AttachmentDecision{TenantID: tenantID, EndpointID: ep.ID, EpEUI: ep.EUI.ToUint64(), Status: status}
}

// requireAttachableKey refuses an endpoint whose network key cannot be sent
// in an attPrp: missing, not Numeric[16] (BSSCI §5.8.1), or zero-filled.
func requireAttachableKey(nwkSnKey []byte) error {
	switch {
	case len(nwkSnKey) == 0:
		return fmt.Errorf("%w: %w", grpcservices.ErrEndpointNotAttachable, ErrMissingNetworkKey)
	case len(nwkSnKey) != endpointKeyLength:
		return fmt.Errorf("%w: %w", grpcservices.ErrEndpointNotAttachable, ErrInvalidNetworkKeyLength)
	case bytes.Equal(nwkSnKey, make([]byte, endpointKeyLength)):
		return fmt.Errorf("%w: %w", grpcservices.ErrEndpointNotAttachable, fmt.Errorf(errFmtKeyZeroFilled, ErrMissingNetworkKey))
	}
	return nil
}

var (
	// ErrEndpointNotFound indicates the requested endpoint does not exist
	ErrEndpointNotFound = errors.New("endpoint not found")

	// ErrMissingNetworkKey indicates endpoint has no network session key
	ErrMissingNetworkKey = errors.New("missing network session key")

	// ErrInvalidNetworkKeyLength indicates network key is not 16 bytes
	ErrInvalidNetworkKeyLength = errors.New("invalid network key length")
)
