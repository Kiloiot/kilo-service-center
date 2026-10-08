package grpc

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// revealedKeyNamesSeparator joins the revealed or removed key names in the event description.
const revealedKeyNamesSeparator = ", "

// endpointKeyFields names each endpoint key by its update mask path.
var endpointKeyFields = []struct {
	name  string
	value func(*models.EndPoint) []byte
}{
	{fieldMaskNwkSnKey, func(ep *models.EndPoint) []byte { return ep.NwkSnKey }},
	{fieldMaskAppKey, func(ep *models.EndPoint) []byte { return ep.AppKey }},
}

// storedEndpointKeys names the keys ep has stored.
func storedEndpointKeys(ep *models.EndPoint) []string {
	var stored []string
	for _, key := range endpointKeyFields {
		if len(key.value(ep)) > 0 {
			stored = append(stored, key.name)
		}
	}
	return stored
}

// removedEndpointKeys names the keys of prior that ep no longer stores.
func removedEndpointKeys(prior []string, ep *models.EndPoint) []string {
	remaining := storedEndpointKeys(ep)
	var removed []string
	for _, name := range prior {
		if !slices.Contains(remaining, name) {
			removed = append(removed, name)
		}
	}
	return removed
}

// revealEndpointKeys puts the requested stored keys of ep into the masked
// result and returns their names; a key that is not stored is not revealed.
func revealEndpointKeys(result *pb.EndPoint, ep *models.EndPoint, requested []pb.EndpointKey) ([]string, error) {
	var revealed []string
	for _, key := range requested {
		switch key {
		case pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY:
			if len(ep.NwkSnKey) > 0 && result.NwkSnKey == nil {
				result.NwkSnKey = ep.NwkSnKey
				revealed = append(revealed, fieldMaskNwkSnKey)
			}
		case pb.EndpointKey_ENDPOINT_KEY_APP_KEY:
			if len(ep.AppKey) > 0 && result.AppKey == nil {
				result.AppKey = ep.AppKey
				revealed = append(revealed, fieldMaskAppKey)
			}
		default:
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidEndpointKeyReveal),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidEndpointKeyReveal))
		}
	}
	return revealed, nil
}

// emitKeysRevealedEvent records who read which keys of the endpoint in clear,
// never the keys, and reports a record it could not write.
func (s *EndpointHandlers) emitKeysRevealedEvent(ctx context.Context, tenantID int64, endpoint *models.EndPoint, revealed []string) error {
	epEui := endpoint.EUI.String()
	return s.keyReveals.RecordRequired(ctx, audit.Event{
		TenantID:    tenantID,
		EventType:   models.EventTypeEndpointKeysRevealed,
		Title:       models.EventTitleEndpointKeysRevealed,
		Description: fmt.Sprintf(models.EventDescriptionEndpointKeysRevealed, epEui, strings.Join(revealed, revealedKeyNamesSeparator)),
		SourceType:  models.SourceTypeEndpoint,
		SourceName:  epEui,
		EndpointID:  &endpoint.ID,
		Details:     map[string]any{bssci.EventKeyEpEui: epEui, models.EventDetailKeyRevealedKeys: revealed},
	})
}

// emitKeysRemovedEvent records who cleared which stored keys of the endpoint; never the keys.
func (s *EndpointHandlers) emitKeysRemovedEvent(ctx context.Context, tenantID int64, endpoint *models.EndPoint, removed []string) {
	epEui := endpoint.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:    tenantID,
		Category:    models.EventCategoryAudit,
		EventType:   models.EventTypeEndpointKeysRemoved,
		Title:       models.EventTitleEndpointKeysRemoved,
		Description: fmt.Sprintf(models.EventDescriptionEndpointKeysRemoved, epEui, strings.Join(removed, revealedKeyNamesSeparator)),
		SourceType:  models.SourceTypeEndpoint,
		SourceName:  epEui,
		EndpointID:  &endpoint.ID,
		Details:     map[string]any{bssci.EventKeyEpEui: epEui, models.EventDetailKeyRemovedKeys: removed},
	})
}
