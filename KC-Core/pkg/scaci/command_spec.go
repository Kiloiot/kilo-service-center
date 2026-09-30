package scaci

import (
	"crypto/x509"
	"fmt"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// routeFunc serves one inbound command. session is a pointer to the
// connection's session slot because connect creates the session.
type routeFunc func(s *Server, conn net.Conn, session **Session, cert *x509.Certificate, opId int64, payload []byte) error

// replayFunc reissues one persisted service-center operation after resume.
type replayFunc func(s *Server, conn net.Conn, session *Session, op *models.SCACIOperation) error

// CommandSpec is one row of the SCACI command table: who initiates the
// command (SCACI §3.2 opId sign), whether an inbound copy is the application
// center completing a handshake (so it does not advance the AC opId
// counter), how it is routed inbound, and how a persisted service-center
// operation of that type is replayed after a session resume (§3.3).
type CommandSpec struct {
	Command   string
	Initiator CommandInitiator
	// HandshakeCompletion marks the responses and completions the application
	// center sends inside a service-center-initiated handshake.
	HandshakeCompletion bool
	// Answers names the service-center request this application center
	// response answers; the response is accepted only for an awaited one.
	Answers string
	// Route serves the command inbound; nil means the service center never
	// accepts it from the application center.
	Route routeFunc
	// Replayable marks service-center operations reissued after resume; the
	// others carry NonReplayReason for the resume log.
	Replayable      bool
	NonReplayReason string
	Replay          replayFunc
}

// sessionRoute adapts a handler that works on an established session.
func sessionRoute(handle func(s *Server, conn net.Conn, session *Session, opId int64) error) routeFunc {
	return func(s *Server, conn net.Conn, session **Session, _ *x509.Certificate, opId int64, _ []byte) error {
		return handle(s, conn, *session, opId)
	}
}

// payloadRoute adapts a handler that decodes a payload on an established session.
func payloadRoute(handle func(s *Server, conn net.Conn, session *Session, opId int64, payload []byte) error) routeFunc {
	return func(s *Server, conn net.Conn, session **Session, _ *x509.Certificate, opId int64, payload []byte) error {
		return handle(s, conn, *session, opId, payload)
	}
}

const (
	nonReplayReasonConnect = "Session establishment handshake per §3.3"
	nonReplayReasonPing    = "Keepalive operation per §3.4, no state change"
	nonReplayReasonStatus  = "AC-initiated per §3.5, SC only responds"
	nonReplayReasonError   = "Error notifications are ephemeral"
)

// commandTable lists every SCACI v1.0.0 command.
var commandTable = []CommandSpec{
	{Command: CmdConnect, Initiator: InitiatorConnect, NonReplayReason: nonReplayReasonConnect,
		Route: func(s *Server, conn net.Conn, session **Session, cert *x509.Certificate, opId int64, payload []byte) error {
			return s.handleConnect(conn, session, cert, opId, payload)
		}},
	{Command: CmdConnectResponse, Initiator: InitiatorConnect, NonReplayReason: nonReplayReasonConnect},
	{Command: CmdConnectComplete, Initiator: InitiatorConnect, NonReplayReason: nonReplayReasonConnect, Route: sessionRoute((*Server).handleConnectComplete)},

	{Command: CmdPing, Initiator: InitiatorEither, NonReplayReason: nonReplayReasonPing, Route: sessionRoute((*Server).handlePing)},
	{Command: CmdPingResponse, Initiator: InitiatorEither, Answers: CmdPing, NonReplayReason: nonReplayReasonPing, Route: sessionRoute((*Server).handlePingResponse)},
	{Command: CmdPingComplete, Initiator: InitiatorEither, HandshakeCompletion: true, NonReplayReason: nonReplayReasonPing, Route: sessionRoute((*Server).handlePingComplete)},

	{Command: CmdStatus, Initiator: InitiatorAC, NonReplayReason: nonReplayReasonStatus, Route: sessionRoute((*Server).handleStatus)},
	{Command: CmdStatusResponse, Initiator: InitiatorAC},
	{Command: CmdStatusComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleStatusComplete)},

	{Command: CmdRegister, Initiator: InitiatorAC, Route: payloadRoute((*Server).handleRegister)},
	{Command: CmdRegisterResponse, Initiator: InitiatorAC},
	{Command: CmdRegisterComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleRegisterComplete)},

	{Command: CmdDeregister, Initiator: InitiatorAC, Route: payloadRoute((*Server).handleDeregister)},
	{Command: CmdDeregisterResponse, Initiator: InitiatorAC},
	{Command: CmdDeregisterComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleDeregisterComplete)},

	{Command: CmdULData, Initiator: InitiatorSC, Replayable: true, Replay: (*Server).replayULData},
	{Command: CmdULDataResponse, Initiator: InitiatorSC, Answers: CmdULData, Route: sessionRoute((*Server).handleULDataResponse)},
	{Command: CmdULDataComplete, Initiator: InitiatorSC, HandshakeCompletion: true, Route: sessionRoute((*Server).rejectACIssuedULDataComplete)},

	{Command: CmdULDataTransmit, Initiator: InitiatorAC, Route: payloadRoute((*Server).handleULDataTransmit)},
	{Command: CmdULDataTransmitResponse, Initiator: InitiatorAC, Route: sessionRoute((*Server).rejectACIssuedULDataTransmitResponse)},
	{Command: CmdULDataTransmitComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleULDataTransmitComplete)},

	{Command: CmdDLDataQueue, Initiator: InitiatorAC, Route: payloadRoute((*Server).handleDLDataQueue)},
	{Command: CmdDLDataQueueResponse, Initiator: InitiatorAC},
	{Command: CmdDLDataQueueComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleDLDataQueueComplete)},

	{Command: CmdDLDataRevoke, Initiator: InitiatorAC, Route: payloadRoute((*Server).handleDLDataRevoke)},
	{Command: CmdDLDataRevokeResponse, Initiator: InitiatorAC},
	{Command: CmdDLDataRevokeComplete, Initiator: InitiatorAC, HandshakeCompletion: true, Route: sessionRoute((*Server).handleDLDataRevokeComplete)},

	{Command: CmdDLDataResult, Initiator: InitiatorSC, Replayable: true, Replay: (*Server).replayDLDataResult},
	{Command: CmdDLDataResultResponse, Initiator: InitiatorSC, HandshakeCompletion: true, Answers: CmdDLDataResult, Route: sessionRoute((*Server).handleDLDataResultResponse)},
	{Command: CmdDLDataResultComplete, Initiator: InitiatorSC, HandshakeCompletion: true, Route: sessionRoute((*Server).rejectACIssuedDLDataResultComplete)},

	{Command: CmdEPStatus, Initiator: InitiatorSC, Replayable: true, Replay: (*Server).replayEPStatus},
	{Command: CmdEPStatusResponse, Initiator: InitiatorSC, Answers: CmdEPStatus, Route: sessionRoute((*Server).handleEPStatusResponse)},
	{Command: CmdEPStatusComplete, Initiator: InitiatorSC, HandshakeCompletion: true, Route: sessionRoute((*Server).rejectACIssuedEPStatusComplete)},

	{Command: CmdError, Initiator: InitiatorEither, NonReplayReason: nonReplayReasonError, Route: payloadRoute((*Server).handleInboundError)},
	{Command: CmdErrorAck, Initiator: InitiatorEither, HandshakeCompletion: true, Route: sessionRoute((*Server).handleErrorAck)},
}

// commandRegistry indexes the command table for one server.
type commandRegistry struct {
	byCommand map[string]*CommandSpec
}

// newCommandRegistry indexes the table; a duplicate row is a programming
// error surfaced at construction.
func newCommandRegistry(table []CommandSpec) (*commandRegistry, error) {
	r := &commandRegistry{byCommand: make(map[string]*CommandSpec, len(table))}
	for i := range table {
		spec := table[i]
		if _, dup := r.byCommand[spec.Command]; dup {
			return nil, fmt.Errorf(errFmtDuplicateCommandSpec, spec.Command)
		}
		r.byCommand[spec.Command] = &spec
	}
	return r, nil
}

func (r *commandRegistry) lookup(command string) (*CommandSpec, bool) {
	spec, ok := r.byCommand[command]
	return spec, ok
}

// handshakeCompletion reports whether an inbound command is the application
// center answering or completing a handshake, which never advances the AC
// opId counter.
func (r *commandRegistry) handshakeCompletion(command string) bool {
	spec, ok := r.byCommand[command]
	return ok && spec.HandshakeCompletion
}
