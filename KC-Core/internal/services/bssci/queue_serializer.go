package bssciservices

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// BSSCI wire envelope keys for serialized response frames.
const (
	wireKeyCommand = "command"
	wireKeyOpId    = "opId" //nolint:revive // BSSCI §2.5 requires lowercase 'd'
)

// queueSerializer implements bssci.QueueSerializer interface
//
// This service builds canonical BSSCI downlink response frames per MIOTY spec.
// All responses follow MessagePack encoding rules and match specification formats.
//
// Spec coverage:
//   - §5.12: DL Data Queue Complete (dlDataQueCmp)
//   - §5.13: DL Data Result Response (dlDataResRsp)
//   - §5.14: DL Data Result Complete (dlDataResCmp)
type queueSerializer struct{}

// NewQueueSerializer creates a new queue serializer service
//
// This is a stateless service - no dependencies required.
func NewQueueSerializer() bssci.QueueSerializer {
	return &queueSerializer{}
}

// BuildDLDataQueueComplete constructs dlDataQueCmp response per BSSCI §5.12
//
// Message Structure:
//   - command (String): "dlDataQueCmp"
//   - opId (Numeric): Operation ID from original dlDataQue request
//
// This message completes the three-way handshake for downlink queue operations.
// The base station has acknowledged the queue request (dlDataQueRsp) and the
// Service Center confirms completion with this message.
func (q *queueSerializer) BuildDLDataQueueComplete(opId int64) map[string]interface{} {
	return map[string]interface{}{
		wireKeyCommand: mioty.CmdDLDataQueueComplete,
		wireKeyOpId:    opId,
	}
}

// BuildDLDataResultResponse constructs dlDataResRsp response per BSSCI §5.14.2
//
// Message Structure:
//   - command (String): "dlDataResRsp"
//   - opId (Numeric): Operation ID from original dlDataRes message
//
// This message acknowledges receipt of a downlink result notification from
// the base station.
func (q *queueSerializer) BuildDLDataResultResponse(opId int64, result *mioty.DLDataResult) map[string]interface{} {
	_ = result // Unused per BSSCI §5.14.2 (queId/success not included in response)
	// Spec §5.14.2: dlDataResRsp MUST contain only command/opId.
	return map[string]interface{}{
		wireKeyCommand: mioty.CmdDLDataResultResponse,
		wireKeyOpId:    opId,
	}
}

// BuildDLDataResultComplete constructs dlDataResCmp response per BSSCI §5.14
//
// Message Structure:
//   - command (String): "dlDataResCmp"
//   - opId (Numeric): Operation ID from original dlDataRes message
//
// This message completes the three-way handshake for downlink result reporting.
// The base station has sent the result (dlDataRes), the Service Center has
// acknowledged (dlDataResRsp), and this message confirms completion.
func (q *queueSerializer) BuildDLDataResultComplete(opId int64) map[string]interface{} {
	return map[string]interface{}{
		wireKeyCommand: mioty.CmdDLDataResultComplete,
		wireKeyOpId:    opId,
	}
}

// BuildDLDataRevokeComplete constructs dlDataRevCmp response per BSSCI §5.13
//
// Message Structure:
//   - command (String): "dlDataRevCmp"
//   - opId (Numeric): Operation ID from original dlDataRev request
//
// This message completes the three-way handshake for downlink revoke operations.
// The base station has acknowledged the revoke request (dlDataRevRsp) and the
// Service Center confirms completion with this message.
func (q *queueSerializer) BuildDLDataRevokeComplete(opID int64) map[string]interface{} {
	return map[string]interface{}{
		wireKeyCommand: mioty.CmdDLDataRevokeComplete,
		wireKeyOpId:    opID,
	}
}
