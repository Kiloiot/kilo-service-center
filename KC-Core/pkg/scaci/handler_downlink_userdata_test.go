package scaci

import (
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// TestHandleDLDataQueue_AcceptsUserDataShapes feeds real MessagePack frames
// through the socket handler: the SCACI §3.10.1 Numeric[m][n] shape and the
// binary entries existing clients send must both reach persistence intact.
func TestHandleDLDataQueue_AcceptsUserDataShapes(t *testing.T) {
	const (
		tenantID int64  = 1
		opID     int64  = 310
		queID    uint64 = 7100001
	)
	wantPayloads := [][]byte{{0x01, 0x02, 0xFF}, {0x10}}

	cases := map[string]interface{}{
		"spec numeric arrays": [][]int{{1, 2, 255}, {16}},
		"binary entries":      wantPayloads,
	}
	for name, userData := range cases {
		t.Run(name, func(t *testing.T) {
			payload, err := msgpack.Marshal(map[string]interface{}{
				"command":   CmdDLDataQueue,
				"opId":      opID,
				"epEui":     uint64(0x70B3D59CD00009E7),
				"queId":     queID,
				"cntDepend": true,
				"packetCnt": []uint32{20, 21},
				"userData":  userData,
			})
			require.NoError(t, err)

			orgID := uuid.New()
			mockDL := new(MockDLService)
			mockEndpoint := new(MockEndpointService)
			mockEndpoint.On("GetByEUI", mock.Anything, tenantID, mock.Anything).Return(&models.EndPoint{Bidi: true}, "")
			mockDL.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
				return dl != nil && assert.ObjectsAreEqual(wantPayloads, dl.UserData) &&
					assert.ObjectsAreEqual(wantPayloads[0], dl.Payload)
			})).Return(&storage.DownlinkMessage{ID: 9, QueID: int64(queID)}, nil)
			mockDL.On("QueueDownlink", mock.Anything, mock.Anything, tenantID, orgID).
				Return(DownlinkQueueOutcome{QueID: queID, Deferred: true}, "")

			server := &Server{
				registry:    newTestRegistry(make(map[net.Conn]*Session), nil),
				codec:       testFrameCodec,
				commands:    mustTestCommandRegistry(),
				clock:       clock.SystemClock{},
				logger:      testLogger(),
				dlSvc:       mockDL,
				endpointSvc: mockEndpoint,
				config:      &Config{},
			}
			conn := &mockConn{}
			session := &Session{TenantID: tenantID, OrganizationID: orgID, State: StateActive}

			require.NoError(t, server.handleDLDataQueue(conn, session, opID, payload))

			var resp mioty.BaseMessage
			require.NoError(t, decodeResponse(conn.written, &resp))
			assert.Equal(t, CmdDLDataQueueResponse, resp.CommandType)
			assert.Equal(t, opID, resp.OpId)
			mockDL.AssertExpectations(t)
		})
	}
}
