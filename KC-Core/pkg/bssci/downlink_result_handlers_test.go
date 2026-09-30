package bssci

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	resultTestTenant  = int64(42)
	resultTestQueueID = uint64(999)
)

// failingResultService answers every dlDataRes with one error.
type failingResultService struct {
	mqttTestDownlinkService
	err error
}

func (d *failingResultService) ProcessDLDataResult(context.Context, *Session, *mioty.DLDataResult) (map[string]interface{}, error) {
	return nil, d.err
}

func newResultServer(t *testing.T) *Server {
	t.Helper()
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, resultTestTenant)
	server.downlinkSvc = &mqttTestDownlinkService{}
	server.tenantResolver.RegisterQueueTenant(int64(resultTestQueueID), strconv.FormatInt(resultTestTenant, 10))
	return server
}

func resultSession(conn *bsscitest.TestConn) *Session {
	return &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "dlresult-station",
			BaseStationEUI:   0xABCDEF1234567890,
			ResolvedTenantID: resultTestTenant,
			DbSessionID:      1,
			Encoding:         EncodingJSON,
		},
		Conn: conn,
	}
}

func dlResultFrame(opID int64) (*Message, map[string]interface{}) {
	return &Message{Command: mioty.CmdDLDataResult, OpId: opID}, map[string]interface{}{
		"epEui":  uint64(0x70B3D59CD00009E6),
		"queId":  resultTestQueueID,
		"result": "expired",
	}
}

// TestHandleDLDataResult_AnswersTheServiceCatalogError: the error frame
// carries the POSIX code and catalog message the service classified.
func TestHandleDLDataResult_AnswersTheServiceCatalogError(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		posix int
		token string
	}{
		{"queue id out of range", NewCatalogError(ErrQueueIDOutOfRange, POSIX_ERANGE), POSIX_ERANGE, errQueueIDOutOfRange},
		{"tenant unresolved", NewCatalogError(ErrCannotResolveTenantForQueue, POSIX_EPROTO), POSIX_EPROTO, errCannotResolveTenantForQueue},
		{"invalid tenant id", NewCatalogError(ErrInvalidTenantIDFormat, POSIX_EINVAL), POSIX_EINVAL, errInvalidTenantIDFormat},
		{"unknown queue id", NewCatalogError(ErrQueueIDNotFound, POSIX_EPROTO), POSIX_EPROTO, errQueueIDNotFound},
		{"unclassified failure", assert.AnError, POSIX_EIO, errDatabaseUpdateFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newResultServer(t)
			server.downlinkSvc = &failingResultService{err: tc.err}
			conn := &bsscitest.TestConn{Encoding: "json"}

			msg, data := dlResultFrame(6103)
			require.NoError(t, server.CallHandleDLDataResult(resultSession(conn), msg, data))

			code, message := conn.LastError()
			assert.Equal(t, tc.posix, code)
			assert.Equal(t, ResolveErrorMessage(tc.token), message)
		})
	}
}
