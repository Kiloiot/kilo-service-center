package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	messagesservice "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/messages"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	receiverTenant        = int64(41)
	receiverForeignTenant = int64(42)
	receiverUplinkID      = "0f3c1c2e-5d7a-4b8e-9a61-2c4d7e9f1a03"
	receiverPrimary       = uint64(0x70B3D59CD00009E6)
	receiverSecondary     = uint64(0x70B3D59CD00009E2)
	receiverUnrelated     = uint64(0x70B3D59CD00009BB)
	receiverPrimaryRssi   = -80.0
	receiverPrimarySnr    = 12.5
	receiverSecondaryRssi = -97.5
	receiverSecondarySnr  = 3.25
	receiverPageSize      = 10
	receiverPoll          = time.Minute
)

// receptionStore holds one tenant's uplinks and lists a station's uplinks as
// the repository does: those it received as the primary receiver or as one of
// the SCACI §3.8.1 receptions.
type receptionStore struct {
	messagesservice.MessageStore
	tenantID int64
	uplinks  []*mioty.ULDataMessage
}

func (s receptionStore) ListByBaseStation(_ context.Context, tenantID int64, bsEui []byte, _ *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	station := mioty.EUI64FromBytes(bsEui)
	var listed []*mioty.ULDataMessage
	for _, uplink := range s.uplinks {
		_, received := uplink.ReceptionAt(station)
		if tenantID == s.tenantID && (uplink.BsEui == station || received) {
			listed = append(listed, uplink)
		}
	}
	return listed, int64(len(listed)), nil
}

func (s receptionStore) GetByID(_ context.Context, tenantID int64, messageID string) (*mioty.ULDataMessage, error) {
	for _, uplink := range s.uplinks {
		if tenantID == s.tenantID && uplink.ID == messageID {
			return uplink, nil
		}
	}
	return nil, storage.ErrNotFound
}

// secondaryReceiverService serves one uplink the primary station and a
// second station both received, through the real message listing service.
func secondaryReceiverService() *CoreService {
	uplink := &mioty.ULDataMessage{
		ID: receiverUplinkID, CommandType: mioty.CmdULData, TenantID: receiverTenant, BsEui: receiverPrimary,
		RxTime: time.Date(2026, time.September, 29, 12, 40, 35, 0, time.UTC).UnixNano(), RSSI: receiverPrimaryRssi, SNR: receiverPrimarySnr,
		BaseStations: []mioty.BaseStationReception{
			{BsEui: receiverPrimary, Rssi: receiverPrimaryRssi, Snr: receiverPrimarySnr},
			{BsEui: receiverSecondary, Rssi: receiverSecondaryRssi, Snr: receiverSecondarySnr},
		},
	}
	listing := messagesservice.New(receptionStore{tenantID: receiverTenant, uplinks: []*mioty.ULDataMessage{uplink}},
		nil, receiverPoll, receiverPoll, streamwake.NewSignal(), 0, logger.NewNop())
	return testCoreService(coreFields{log: testLogger{}, msgListingSvc: listing})
}

// A station that received an uplink as a secondary receiver lists it and
// retrieves the same message, both with its own radio measurements.
func TestGetBaseStationMessage_ASecondaryReceiverRetrievesTheUplinkItLists(t *testing.T) {
	svc := secondaryReceiverService()
	ctx := testutil.TestContextWithTenant(receiverTenant)
	secondary := mioty.FormatEUI64(receiverSecondary)

	listed, err := svc.ListBaseStationMessages(ctx, &pb.ListBaseStationMessagesRequest{BsEui: secondary, PageSize: receiverPageSize})
	require.NoError(t, err)
	require.Len(t, listed.Messages, 1)
	assert.Equal(t, receiverSecondaryRssi, listed.Messages[0].Rssi)

	got, err := svc.GetBaseStationMessage(ctx, &pb.GetBaseStationMessageRequest{BsEui: secondary, MessageId: listed.Messages[0].Id})
	require.NoError(t, err)
	assert.Equal(t, secondary, got.Message.BsEui)
	assert.Equal(t, receiverSecondaryRssi, got.Message.Rssi)
	assert.Equal(t, receiverSecondarySnr, got.Message.Snr)

	primary, err := svc.GetBaseStationMessage(ctx, &pb.GetBaseStationMessageRequest{BsEui: mioty.FormatEUI64(receiverPrimary), MessageId: receiverUplinkID})
	require.NoError(t, err)
	assert.Equal(t, receiverPrimaryRssi, primary.Message.Rssi)
	assert.Equal(t, receiverPrimarySnr, primary.Message.Snr)
}

// A station that did not receive the uplink, and another tenant, cannot
// retrieve it.
func TestGetBaseStationMessage_RefusesAStationThatDidNotReceiveItAndAnotherTenant(t *testing.T) {
	svc := secondaryReceiverService()

	for name, request := range map[string]struct {
		ctx     context.Context
		station uint64
	}{
		"unrelated station": {testutil.TestContextWithTenant(receiverTenant), receiverUnrelated},
		"foreign tenant":    {testutil.TestContextWithTenant(receiverForeignTenant), receiverSecondary},
	} {
		_, err := svc.GetBaseStationMessage(request.ctx, &pb.GetBaseStationMessageRequest{BsEui: mioty.FormatEUI64(request.station), MessageId: receiverUplinkID})
		assert.Equal(t, codes.NotFound, status.Code(err), name)
	}
}
