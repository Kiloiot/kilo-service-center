package bssci

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// operationRole places an inbound command in the operation sequencing rules
// (rev1 §5.2 / classic §3.2): the base station opens its own operations with
// strictly increasing positive IDs and completes them with the same ID, and it
// answers service-center operations with their negative ID.
type operationRole int

const (
	// roleUnset is the zero value: a table row must name its role, so the
	// registry refuses it.
	roleUnset operationRole = iota
	// roleServiceCenterOnly marks commands only the service center sends; an
	// inbound copy is refused by the direction check, not by sequencing.
	roleServiceCenterOnly
	// roleHandshake belongs to the connect operation (opId 0) and is legal
	// only until that operation completes.
	roleHandshake
	// roleOpens starts a base-station operation.
	roleOpens
	// roleCloses completes an open base-station operation.
	roleCloses
	// roleResponds answers an operation the service center started.
	roleResponds
	// roleErrorExchange replaces the rest of the operation whose ID it
	// carries, in either direction and in any phase (rev1 §5.17).
	roleErrorExchange
)

// admits checks the role against the command's direction: a command only the
// service center sends has no inbound sequencing, every other one needs the
// role its inbound copy is sequenced by.
func (r operationRole) admits(direction CommandDirection) error {
	switch {
	case r == roleUnset:
		return errCommandSpecWithoutRole
	case (r == roleServiceCenterOnly) != (direction == DirectionSCtoBS):
		return errCommandSpecRoleContradictsDirection
	}
	return nil
}

// sequenced reports whether the role is subject to operation ID sequencing.
func (r operationRole) sequenced() bool {
	return r == roleOpens || r == roleCloses || r == roleResponds
}

// CommandSpec is one row of the BSSCI command table: how an inbound command
// is dispatched and how a service-center-issued operation of that type is
// normalized, rebuilt and resumed after a reconnect.
type CommandSpec struct {
	Command string
	// Role places the command in operation sequencing when a base station
	// sends it.
	Role operationRole
	// Handler serves the command when a base station sends it; nil for
	// commands only the service center issues.
	Handler HandlerFunc
	// Normalize repairs numeric types after a persisted operation is decoded
	// from JSON; nil means only opId needs repair.
	Normalize func(msg map[string]interface{}) map[string]interface{}
	// Reconstitute rebuilds the wire message of a persisted operation from
	// its sanitized copy and metadata; nil means the persisted copy is complete.
	Reconstitute func(s *Server, msg, metadata map[string]interface{}, op *PendingOperation) (map[string]interface{}, error)
	// Resumable marks service-center operations that are reissued after a
	// session resume (BSSCI §3.3 / §5.2).
	Resumable bool
	// OnError undoes the domain effect of a service-center operation the base
	// station answered with error (BSSCI §3.17); nil means there is none.
	OnError func(s *Server, ctx context.Context, session *Session, opID int64, code int, message string)
}

// commandTable lists every BSSCI command the server handles or issues.
// Response and complete rows exist for the base-station-initiated half of
// each handshake; service-center-initiated operations carry the resume rows.
var commandTable = []CommandSpec{
	{Command: mioty.CmdConnect, Role: roleHandshake, Handler: (*Server).handleConnect},
	{Command: mioty.CmdConnectComplete, Role: roleHandshake, Handler: (*Server).handleConnectComplete},
	{Command: mioty.CmdPing, Role: roleOpens, Handler: (*Server).handlePing},
	{Command: mioty.CmdPingResponse, Role: roleResponds, Handler: (*Server).handlePingResponse},
	{Command: mioty.CmdPingComplete, Role: roleCloses, Handler: (*Server).handlePingComplete},
	{Command: mioty.CmdStatus, Role: roleServiceCenterOnly, Resumable: true},
	{Command: mioty.CmdStatusResponse, Role: roleResponds, Handler: (*Server).handleStatusResponse},
	{Command: mioty.CmdAttach, Role: roleOpens, Handler: (*Server).handleAttach},
	{Command: mioty.CmdAttachComplete, Role: roleCloses, Handler: (*Server).handleAttachComplete},
	{Command: mioty.CmdDetach, Role: roleOpens, Handler: (*Server).handleDetach},
	{Command: mioty.CmdDetachComplete, Role: roleCloses, Handler: (*Server).handleDetachComplete},
	{Command: mioty.CmdULData, Role: roleOpens, Handler: (*Server).handleULData},
	{Command: mioty.CmdULDataComplete, Role: roleCloses, Handler: (*Server).handleULDataComplete},
	{
		Command:   mioty.CmdULDataTransmit,
		Role:      roleServiceCenterOnly,
		Resumable: true,
		Reconstitute: func(s *Server, msg, metadata map[string]interface{}, op *PendingOperation) (map[string]interface{}, error) {
			return s.reconstitueULDataTxMessage(msg, metadata, op)
		},
	},
	{Command: mioty.CmdULDataTransmitResponse, Role: roleResponds, Handler: (*Server).handleULDataTxResponse},
	{Command: mioty.CmdError, Role: roleErrorExchange, Handler: (*Server).handleError},
	{Command: mioty.CmdErrorAck, Role: roleErrorExchange, Handler: (*Server).handleErrorAck},
	{
		Command:   mioty.CmdAttachPropagate,
		Role:      roleServiceCenterOnly,
		Resumable: true,
		Normalize: normalizeAttPrpMessage,
		Reconstitute: func(s *Server, msg, metadata map[string]interface{}, _ *PendingOperation) (map[string]interface{}, error) {
			return s.reconstitueAttachPropagateMessage(msg, metadata)
		},
	},
	{Command: mioty.CmdAttachPropagateResponse, Role: roleResponds, Handler: (*Server).handleAttachPropagateResponse},
	{Command: mioty.CmdDetachPropagate, Role: roleServiceCenterOnly, Resumable: true, Normalize: normalizeDetPrpMessage},
	{Command: mioty.CmdDetachPropagateResponse, Role: roleResponds, Handler: (*Server).handleDetachPropagateResponse},
	{Command: mioty.CmdDLDataResult, Role: roleOpens, Handler: (*Server).handleDLDataResult},
	{Command: mioty.CmdDLDataResultResponse, Role: roleServiceCenterOnly, Handler: (*Server).handleDLDataResultResponse},
	{Command: mioty.CmdDLDataResultComplete, Role: roleCloses, Handler: (*Server).handleDLDataResultComplete},
	{Command: mioty.CmdDLRxStatus, Role: roleOpens, Handler: (*Server).handleDLRXStatus},
	{Command: mioty.CmdDLRxStatusResponse, Role: roleServiceCenterOnly, Handler: (*Server).handleDLRXStatusResponse},
	{Command: mioty.CmdDLRxStatusComplete, Role: roleCloses, Handler: (*Server).handleDLRXStatusComplete},
	{Command: mioty.CmdDLRxStatusQuery, Role: roleServiceCenterOnly, Resumable: true},
	{Command: mioty.CmdDLRxStatusQueryResponse, Role: roleResponds, Handler: (*Server).handleDLRXStatusQueryResponse},
	{
		Command:   mioty.CmdDLDataRevoke,
		Role:      roleServiceCenterOnly,
		Resumable: true,
		Normalize: normalizeDlDataRevMessage,
		Reconstitute: func(s *Server, msg, metadata map[string]interface{}, _ *PendingOperation) (map[string]interface{}, error) {
			return s.reconstitueDLDataRevMessage(msg, metadata)
		},
		OnError: (*Server).recordRevokeRefusal,
	},
	{Command: mioty.CmdDLDataRevokeResponse, Role: roleResponds, Handler: (*Server).handleDLDataRevokeResponse},
	{
		Command:   mioty.CmdDLDataQueue,
		Role:      roleServiceCenterOnly,
		Resumable: true,
		Reconstitute: func(s *Server, msg, metadata map[string]interface{}, op *PendingOperation) (map[string]interface{}, error) {
			return s.reconstitueDLDataQueMessage(msg, metadata, op)
		},
		OnError: (*Server).rejectQueuedDownlink,
	},
	{Command: mioty.CmdDLDataQueueResponse, Role: roleResponds, Handler: (*Server).handleDLDataQueueResponse},
	{Command: mioty.CmdVMActivate, Role: roleServiceCenterOnly, Handler: (*Server).handleVMActivate},
	{Command: mioty.CmdVMActivateResponse, Role: roleResponds, Handler: (*Server).handleVMActivateResponse},
	{Command: mioty.CmdVMActivateComplete, Role: roleResponds, Handler: (*Server).handleVMActivateComplete},
	{Command: mioty.CmdVMDeactivate, Role: roleServiceCenterOnly, Handler: (*Server).handleVMDeactivate},
	{Command: mioty.CmdVMDeactivateResponse, Role: roleResponds, Handler: (*Server).handleVMDeactivateResponse},
	{Command: mioty.CmdVMDeactivateComplete, Role: roleResponds, Handler: (*Server).handleVMDeactivateComplete},
	{Command: mioty.CmdVMStatus, Role: roleServiceCenterOnly, Handler: (*Server).handleVMStatus},
	{Command: mioty.CmdVMStatusResponse, Role: roleResponds, Handler: (*Server).handleVMStatusResponse},
	{Command: mioty.CmdVMStatusComplete, Role: roleResponds, Handler: (*Server).handleVMStatusComplete},
	{Command: mioty.CmdVMDLData, Role: roleServiceCenterOnly, Handler: (*Server).handleVMDLData},
	{Command: mioty.CmdVMDLDataResponse, Role: roleResponds, Handler: (*Server).handleVMDLDataResponse},
	{Command: mioty.CmdVMDLDataComplete, Role: roleResponds, Handler: (*Server).handleVMDLDataComplete},
}

// commandRegistry is a server's copy of the command table keyed by command,
// paired with the protocol direction of every known command.
type commandRegistry struct {
	byCommand map[string]*CommandSpec
	direction map[string]CommandDirection
}

// newCommandRegistry indexes the table; a duplicate row, a command the
// direction map does not know and a row without a role, or with a role its
// direction contradicts, are programming errors surfaced at startup.
func newCommandRegistry(table []CommandSpec, directions map[string]CommandDirection) (*commandRegistry, error) {
	r := &commandRegistry{
		byCommand: make(map[string]*CommandSpec, len(table)),
		direction: directions,
	}
	for i := range table {
		spec := table[i]
		if _, dup := r.byCommand[spec.Command]; dup {
			return nil, fmt.Errorf(errFmtDuplicateCommandSpec, spec.Command)
		}
		direction, known := directions[spec.Command]
		if !known {
			return nil, fmt.Errorf(errFmtCommandWithoutDirection, spec.Command)
		}
		if err := spec.Role.admits(direction); err != nil {
			return nil, fmt.Errorf(errFmtCommandSpecRole, err, spec.Command)
		}
		r.byCommand[spec.Command] = &spec
	}
	return r, nil
}

func (r *commandRegistry) lookup(command string) (*CommandSpec, bool) {
	spec, ok := r.byCommand[command]
	return spec, ok
}

// inboundHandler returns the handler for a command a base station may send.
func (r *commandRegistry) inboundHandler(command string) (HandlerFunc, bool) {
	spec, ok := r.byCommand[command]
	if !ok || spec.Handler == nil {
		return nil, false
	}
	return spec.Handler, true
}

// isServiceCenterCommand reports whether only the service center sends the
// command, so an inbound copy is a protocol violation.
func (r *commandRegistry) isServiceCenterCommand(command string) bool {
	direction, known := r.direction[command]
	return known && direction == DirectionSCtoBS
}

// shouldNormalize reports whether an inbound payload is normalized: only
// commands a base station may send, and only known ones (BSSCI §2.4).
func (r *commandRegistry) shouldNormalize(command string) bool {
	direction, known := r.direction[command]
	return known && (direction == DirectionBStoSC || direction == DirectionBidirectional)
}

// role returns how a base station's copy of the command is sequenced.
func (r *commandRegistry) role(command string) operationRole {
	if spec, ok := r.byCommand[command]; ok {
		return spec.Role
	}
	return roleUnset
}

// resumable reports whether a persisted service-center operation of this type
// is reissued after a session resume.
func (r *commandRegistry) resumable(opID int64, command string) bool {
	if opID >= 0 {
		return false
	}
	spec, ok := r.byCommand[command]
	return ok && spec.Resumable
}

// onError returns the compensation for an errored operation of this type.
func (r *commandRegistry) onError(command string) (func(s *Server, ctx context.Context, session *Session, opID int64, code int, message string), bool) {
	spec, ok := r.byCommand[command]
	if !ok || spec.OnError == nil {
		return nil, false
	}
	return spec.OnError, true
}

// normalize repairs the numeric types of a persisted operation message.
func (r *commandRegistry) normalize(msg map[string]interface{}, command string) map[string]interface{} {
	if spec, ok := r.byCommand[command]; ok && spec.Normalize != nil {
		return spec.Normalize(msg)
	}
	return normalizeOpIDOnly(msg)
}

// normalizeOpIDOnly copies the message and repairs the opId type, the one
// field every persisted operation carries.
func normalizeOpIDOnly(msg map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(msg))
	for k, v := range msg {
		normalized[k] = v
	}
	if opId, ok := msg["opId"].(float64); ok {
		normalized["opId"] = int64(opId)
	}
	return normalized
}
