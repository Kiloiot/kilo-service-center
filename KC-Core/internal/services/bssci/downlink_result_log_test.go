package bssciservices

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	resultLogQueueID = uint64(900)
	resultLogTxTime  = int64(1700000000123456789)
)

func processLoggedResult(t *testing.T, result *mioty.DLDataResult) *bsscitest.RecordingLogger {
	t.Helper()
	log := bsscitest.NewRecordingLogger()
	resolver := NewTenantResolver(nil)
	resolver.RegisterQueueTenant(int64(resultLogQueueID), "3")
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: log, Tenants: resolver, Outcomes: &mockMIOTYDownlinksForDispatch{}, Holders: &mockMIOTYDownlinksForDispatch{},
		Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)
	session := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70b3d59cd00009e6}}

	_, err = svc.ProcessDLDataResult(testutil.TestContext(), session, result)
	require.NoError(t, err)
	return log
}

// TestProcessDLDataResult_LogsTheReportedValues: the optional txTime and
// packetCnt are logged as values, not as the addresses of the fields.
func TestProcessDLDataResult_LogsTheReportedValues(t *testing.T) {
	txTime := resultLogTxTime
	packetCnt := uint32(17)
	log := processLoggedResult(t, &mioty.DLDataResult{
		EpEui: 0x70b3d59cd0000341, QueId: resultLogQueueID, Result: mioty.ResultSent,
		TxTime: &txTime, PacketCnt: &packetCnt,
	})

	for _, msg := range []string{bssci.LogBSSCIReceivedDLDataResFromBaseStation, bssci.LogBSSCIUpdatedDownlinkResult} {
		entries := log.FilterMessage(msg)
		require.Len(t, entries, 1, msg)
		fields := entries[0].FieldMap()
		assert.Equal(t, resultLogTxTime, fields[logger.FieldTxTime], msg)
	}
	fields := log.FilterMessage(bssci.LogBSSCIReceivedDLDataResFromBaseStation)[0].FieldMap()
	assert.Equal(t, uint32(17), fields[logger.FieldPacketCnt])
}

// TestProcessDLDataResult_OmitsAbsentOptionalFields: an expired downlink
// carries neither txTime nor packetCnt (BSSCI §3.14.1), so neither is logged.
func TestProcessDLDataResult_OmitsAbsentOptionalFields(t *testing.T) {
	log := processLoggedResult(t, &mioty.DLDataResult{
		EpEui: 0x70b3d59cd0000341, QueId: resultLogQueueID, Result: mioty.ResultExpired,
	})

	for _, msg := range []string{bssci.LogBSSCIReceivedDLDataResFromBaseStation, bssci.LogBSSCIUpdatedDownlinkResult} {
		entries := log.FilterMessage(msg)
		require.Len(t, entries, 1, msg)
		fields := entries[0].FieldMap()
		assert.NotContains(t, fields, logger.FieldTxTime, msg)
		assert.NotContains(t, fields, logger.FieldPacketCnt, msg)
	}
}
