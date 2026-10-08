package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// A resumed session's conRsp carries only the fields rev1 §5.3.2 defines:
// the operation counters live in the con message, never in its response
// (§4.5 forbids extra fields).
func TestResumeConnectResponseCarriesOnlySpecFields(t *testing.T) {
	server := newResumeReissueServer(t)
	sessionSvc := server.sessionSvc.(*mockSessionService)

	prevUUID := make([]byte, 16)
	snBsUUID := make([]interface{}, 16)
	for i := range prevUUID {
		prevUUID[i] = byte(i + 1)
		snBsUUID[i] = int64(i + 1)
	}
	sessionSvc.StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
		ID:             "previous-runtime-session",
		BaseStationEUI: TestBsEui01,
		SessionUUID:    prevUUID,
		DbSessionID:    7,
		LastBsOpId:     12,
		LastScOpId:     -4,
	}})

	conn := &testutil.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "resuming-connection", BaseStationEUI: TestBsEui01, Encoding: EncodingJSON},
		Conn:                 conn,
	}
	connectData := map[string]interface{}{
		"command":  mioty.CmdConnect,
		"opId":     int64(0),
		"version":  mioty.MIOTYProtocolVersion,
		"bsEui":    int64(TestBsEui01),
		"bidi":     true,
		"snBsUuid": snBsUUID,
	}
	require.NoError(t, server.handleConnect(session, &Message{OpId: 0, Command: mioty.CmdConnect, Data: connectData}, connectData))

	require.Len(t, conn.SentMessages, 1)
	conRsp := conn.SentMessages[0]
	require.Equal(t, mioty.CmdConnectResponse, conRsp["command"])
	assert.Equal(t, true, conRsp["snResume"], "the prior session is resumed")

	specFields := map[string]bool{
		"command": true, "opId": true, "version": true, "scEui": true, "vendor": true, "model": true,
		"name": true, "swVersion": true, "info": true, "snResume": true, "snScUuid": true,
	}
	for field := range conRsp {
		assert.True(t, specFields[field], "conRsp carries %q, which rev1 §5.3.2 does not define", field)
	}
}

// The outbound validator refuses a conRsp that carries the con-only counters.
func TestOutboundValidationRejectsCountersInConnectResponse(t *testing.T) {
	server := newResumeReissueServer(t)
	session := &Session{ProtocolSessionState: ProtocolSessionState{ID: "validation-session", Encoding: EncodingJSON}}
	for _, counter := range []string{"snBsOpId", "snScOpId"} {
		payload := buildMinimalValidPayload(mioty.CmdConnectResponse)
		payload[counter] = int64(1)
		err := server.validateOutboundMessage(session, payload)
		var catalogErr *CatalogError
		require.ErrorAs(t, err, &catalogErr, counter)
		assert.Equal(t, errOutboundExtraField, catalogErr.Token, counter)
	}
}
