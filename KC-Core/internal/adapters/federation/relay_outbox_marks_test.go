package federation

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	federationpb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/federation/v1"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var errOutboxDown = errors.New("outbox down")

// failingMarksOutbox refuses every status change.
type failingMarksOutbox struct {
	interfaces.FederationOutboxRepository
}

func (failingMarksOutbox) MarkSent(context.Context, uuid.UUID) error { return errOutboxDown }
func (failingMarksOutbox) MarkAcked(context.Context, uuid.UUID) error {
	return errOutboxDown
}
func (failingMarksOutbox) MarkRejected(context.Context, uuid.UUID, string) error {
	return errOutboxDown
}

// sendingStream accepts every message the relay writes.
type sendingStream struct {
	federationpb.FederationService_ConnectClient
	sent int
}

func (s *sendingStream) Send(*federationpb.CEToECEMessage) error {
	s.sent++
	return nil
}

func newMarksTestClient(log *bsscitest.RecordingLogger) *RelayClient {
	return NewRelayClient(config.FederationConfig{}, &relayInstallRepo{inst: onboardedInstall()}, failingMarksOutbox{}, log)
}

func TestRelayRecord_UnmarkedRecordEndsTheStream(t *testing.T) {
	client := newMarksTestClient(bsscitest.NewRecordingLogger())
	stream := &sendingStream{}

	err := client.relayRecord(testutil.TestContext(), stream, &models.FederationOutboxRecord{RelayID: uuid.New()})

	require.ErrorIs(t, err, ErrOutboxMarkSent, "a record left pending would be resent on every drain pass")
	require.ErrorIs(t, err, errOutboxDown)
	assert.Equal(t, 1, stream.sent)
}

func TestHandleReceipt_UnrecordedReceiptIsLogged(t *testing.T) {
	for _, accepted := range []bool{true, false} {
		log := bsscitest.NewRecordingLogger()
		client := newMarksTestClient(log)

		client.handleReceipt(testutil.TestContext(), &federationpb.UplinkReceipt{RelayId: uuid.NewString(), Accepted: accepted})

		assert.Len(t, log.FilterMessage(LogRelayReceiptNotRecorded), 1, "accepted=%v", accepted)
	}
}
