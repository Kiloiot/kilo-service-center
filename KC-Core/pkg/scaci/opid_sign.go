// Package scaci provides opId sign classification per SCACI §3.2
package scaci

// CommandInitiator defines who can initiate a command per SCACI §3.2
type CommandInitiator string

const (
	// InitiatorAC indicates command is AC-initiated (positive opId required)
	InitiatorAC CommandInitiator = "ac"
	// InitiatorSC indicates command is SC-initiated (negative opId required)
	InitiatorSC CommandInitiator = "sc"
	// InitiatorEither indicates command can be initiated by either party (any sign valid)
	InitiatorEither CommandInitiator = "either"
	// InitiatorConnect indicates Connect handshake (opId=0 only, validated separately)
	InitiatorConnect CommandInitiator = "connect"
)

// CommandInitiatorMap classifies SCACI commands by allowed initiator per §3.2
//
// Spec alignment:
//   - Connect/ConRsp/ConCmp → opId=0 only (§3.3); ConnectResponse never arrives inbound
//   - Ping/PingRsp/PingCmp → either party can initiate (§3.4)
//   - Status/Reg/Dereg/DLDataQueue/DLDataRevoke → AC-initiated, positive opId
//   - ULDataTx/ULDataTxRsp/ULDataTxCmp → AC-initiated flow, all use same positive opId (§3.9)
//   - ULData/ULDataRsp/ULDataCmp → SC-initiated, negative opId (§3.8)
//   - DLDataResult/DLDataResultRsp/DLDataResultCmp → SC-initiated flow, negative opId (§3.12)
//   - EPStatus/EPStatusRsp/EPStatusCmp → SC-initiated flow, negative opId (§3.13)
//   - Error/ErrorAck → either, echoes triggering opId (§3.14)
var CommandInitiatorMap = map[string]CommandInitiator{
	// Connect (§3.3) - opId=0 only
	CmdConnect:         InitiatorConnect,
	CmdConnectResponse: InitiatorConnect,
	CmdConnectComplete: InitiatorConnect,

	// Ping (§3.4) - Either party can initiate
	CmdPing:         InitiatorEither,
	CmdPingResponse: InitiatorEither,
	CmdPingComplete: InitiatorEither,

	// Status (§3.5) - AC-initiated, positive opId
	CmdStatus:         InitiatorAC,
	CmdStatusResponse: InitiatorAC,
	CmdStatusComplete: InitiatorAC,

	// Register (§3.6) - AC-initiated, positive opId
	CmdRegister:         InitiatorAC,
	CmdRegisterResponse: InitiatorAC,
	CmdRegisterComplete: InitiatorAC,

	// Deregister (§3.7) - AC-initiated, positive opId
	CmdDeregister:         InitiatorAC,
	CmdDeregisterResponse: InitiatorAC,
	CmdDeregisterComplete: InitiatorAC,

	// UL Data (§3.8) - SC-initiated, negative opId
	CmdULData:         InitiatorSC,
	CmdULDataResponse: InitiatorSC,
	CmdULDataComplete: InitiatorSC,

	// UL Data Transmit (§3.9) - AC-initiated flow, SC responds with same positive opId
	CmdULDataTransmit:         InitiatorAC,
	CmdULDataTransmitResponse: InitiatorAC,
	CmdULDataTransmitComplete: InitiatorAC,

	// DL Data Queue (§3.10) - AC-initiated, positive opId
	CmdDLDataQueue:         InitiatorAC,
	CmdDLDataQueueResponse: InitiatorAC,
	CmdDLDataQueueComplete: InitiatorAC,

	// DL Data Revoke (§3.11) - AC-initiated, positive opId
	CmdDLDataRevoke:         InitiatorAC,
	CmdDLDataRevokeResponse: InitiatorAC,
	CmdDLDataRevokeComplete: InitiatorAC,

	// DL Data Result (§3.12) - SC-initiated, negative opId
	CmdDLDataResult:         InitiatorSC,
	CmdDLDataResultResponse: InitiatorSC,
	CmdDLDataResultComplete: InitiatorSC,

	// EP Status (§3.13) - SC-initiated, negative opId
	CmdEPStatus:         InitiatorSC,
	CmdEPStatusResponse: InitiatorSC,
	CmdEPStatusComplete: InitiatorSC,

	// Error (§3.14) - echoes triggering opId, either sign valid
	CmdError:    InitiatorEither,
	CmdErrorAck: InitiatorEither,
}
