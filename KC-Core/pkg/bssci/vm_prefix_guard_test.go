package bssci

import (
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/stretchr/testify/assert"
)

// testConn implements net.Conn for testing without panics
type testConn struct {
	net.Conn
	errorSent bool
}

func (d *testConn) Write(p []byte) (n int, err error) {
	d.errorSent = true
	return len(p), nil
}

func (d *testConn) Close() error                       { return nil }
func (d *testConn) Read(_ []byte) (n int, err error)   { return 0, nil }
func (d *testConn) RemoteAddr() net.Addr               { return nil }
func (d *testConn) LocalAddr() net.Addr                { return nil }
func (d *testConn) SetDeadline(_ time.Time) error      { return nil }
func (d *testConn) SetReadDeadline(_ time.Time) error  { return nil }
func (d *testConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestVMPrefixGuardAllowsRegisteredHandlers(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "VM prefix allowed",
			command:       "vm.test",
			shouldSucceed: true,
			description:   "Registered vm.* handler should be invoked",
		},
		{
			name:          "RC prefix allowed",
			command:       "rc.test",
			shouldSucceed: true,
			description:   "Registered rc.* handler should be invoked",
		},
		{
			name:          "Unknown prefix rejected",
			command:       "xyz.test",
			shouldSucceed: false,
			description:   "Unsupported sublayer prefix returns POSIX_ENOTSUP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use NewTestServerWithMemoryStatusService (StatusService is mandatory)
			s := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
			s.handlers = make(map[string]HandlerFunc)
			s.config = &Config{MessageEncoding: EncodingJSON}

			// Create test connection
			conn := &testConn{}
			session := &Session{
				HandshakeComplete: true,
				Conn:              conn,
				Encoding:          EncodingJSON,
			}

			// Register handler that flips boolean
			handlerCalled := false
			testHandler := func(_ *Server, _ *Session, _ *Message, _ map[string]interface{}) error {
				handlerCalled = true
				return nil
			}
			s.handlers[tt.command] = testHandler

			// Create message
			msg := &Message{
				Command: tt.command,
				OpId:    0,
			}
			data := map[string]interface{}{
				"command": tt.command,
				"opId":    float64(0),
			}

			// Call handleMessage
			_ = s.handleMessage(session, msg, data)

			if tt.shouldSucceed {
				assert.True(t, handlerCalled, "Handler should have been invoked for %s", tt.command)
				assert.False(t, conn.errorSent, "No error should be sent for allowed prefix")
			} else {
				assert.False(t, handlerCalled, "Handler should not be invoked for unsupported prefix")
				assert.True(t, conn.errorSent, "Error should be sent for unsupported prefix")
			}
		})
	}
}
