package bssciservices

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// endpointManager holds only the role that sees the downlink queue.
var endpointManager = authz.Roles{EndpointManager: true}

// storedAuditDownlink is the owner's downlink as the queue returns it.
func storedAuditDownlink() *storage.DownlinkMessage {
	return &storage.DownlinkMessage{EPEUI: auditEndpointText, TenantID: auditOwnerTenant, QueID: auditQueueID, Payload: []byte{0x33}}
}

// Every change of the service center queue is announced to the endpoint
// managers who see that queue: in their category, under the owner tenant,
// naming the endpoint and the queue id so each open view of the endpoint's
// queue refreshes.
func TestDownlinkAuditLog_AnnouncesEveryQueueChangeToEndpointManagers(t *testing.T) {
	cases := map[string]struct {
		record      func(*DownlinkAuditLog) error
		eventType   string
		title       string
		description string
		station     bool
	}{
		"revoked in the queue": {
			record: func(a *DownlinkAuditLog) error {
				return a.RecordQueueRevoked(testutil.TestContext(), storedAuditDownlink())
			},
			eventType:   models.EventTypeDLDataRevoked,
			title:       "Downlink revoked",
			description: "Downlink 118 for endpoint 70B3D56770111505 revoked in the service center queue before any base station held it, payload 33",
		},
		"queued at the service center": {
			record: func(a *DownlinkAuditLog) error {
				return a.RecordEnqueued(testutil.TestContext(), storedAuditDownlink())
			},
			eventType:   models.EventTypeDLDataEnqueued,
			title:       "Downlink queued at service center",
			description: "Downlink 118 for endpoint 70B3D56770111505 queued at the service center, payload 33",
		},
		"updated while pending": {
			record: func(a *DownlinkAuditLog) error {
				return a.RecordPendingUpdated(testutil.TestContext(), storedAuditDownlink())
			},
			eventType:   models.EventTypeDLDataUpdated,
			title:       "Pending downlink updated",
			description: "Pending downlink 118 for endpoint 70B3D56770111505 updated in the service center queue, payload 33",
		},
		"returned to the queue by its station": {
			record: func(a *DownlinkAuditLog) error {
				return a.RecordRequeued(testutil.TestContext(), storage.PendingDownlink{
					QueID: uint64(auditQueueID), TenantID: auditOwnerTenantID, OrganizationID: uuid.New(), EpEUI: auditEndpointEUI,
				}, auditStationEUI)
			},
			eventType: models.EventTypeDLDataRequeued,
			title:     "Downlink returned to the queue",
			description: "Downlink 118 for endpoint 70B3D56770111505 returned to the service center queue: base station " +
				auditStationLabel + " no longer holds it, payload 33",
			station: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			events := &mockEventStore{}
			audit := mustAuditLogger(t, events, queuedDownlinks{downlink: storedAuditDownlink()}, ownersStation)

			require.NoError(t, tc.record(audit))

			event := events.lastEvent
			require.NotNil(t, event)
			assert.Equal(t, tc.eventType, event.EventType)
			assert.Equal(t, models.EventCategoryMessage, event.Category)
			assert.True(t, authz.CanReadCategory(endpointManager, event.Category), "an endpoint manager's stream carries it")
			assert.Equal(t, auditOwnerTenant, event.TenantID)
			assert.Equal(t, tc.title, event.Title)
			assert.Equal(t, tc.description, event.Description)
			assert.Equal(t, auditEndpointText, event.SourceName)
			details := eventDetails(t, event)
			assert.Equal(t, auditEndpointText, details[models.EventDetailKeyEpEui])
			assert.Equal(t, json.Number("118"), details[models.EventDetailKeyQueID])
			if tc.station {
				assert.Equal(t, "70B3D59CD00009E6", details[models.EventDetailKeyBsEui], "the station that let it go")
			} else {
				assert.NotContains(t, details, models.EventDetailKeyBsEui, "no base station holds it")
			}
		})
	}
}

// A downlink whose owner the queue row does not name is refused, not filed
// under another tenant.
func TestDownlinkAuditLog_RefusesADownlinkWithoutItsOwner(t *testing.T) {
	events := &mockEventStore{}
	audit := mustAuditLogger(t, events, queuedDownlinks{}, ownersStation)
	ownerless := storedAuditDownlink()
	ownerless.TenantID = ""

	for _, record := range []func(context.Context, *storage.DownlinkMessage) error{
		audit.RecordQueueRevoked, audit.RecordEnqueued, audit.RecordPendingUpdated,
	} {
		require.ErrorIs(t, record(testutil.TestContext(), ownerless), errInvalidOwnerTenant)
	}
	assert.Nil(t, events.lastEvent)
}
