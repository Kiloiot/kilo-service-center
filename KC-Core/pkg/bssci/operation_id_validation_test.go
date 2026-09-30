package bssci

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
)

// sequencingTestEUI marks a session whose con was already processed.
const sequencingTestEUI = 123456789

func newSequencingTestServer() *Server {
	return NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
}

// TestSCOperationIDMustBeNegative verifies BSSCI §3.2-02: a response to a
// service-center operation carries that operation's negative ID.
func TestSCOperationIDMustBeNegative(t *testing.T) {
	tests := []struct {
		name     string
		opId     int64
		expected string
	}{
		{name: "PositiveResponseIDRejected", opId: 1, expected: errSCOperationIDMustBeNegative},
		{name: "ZeroResponseIDRejected", opId: 0, expected: errSCOperationIDMustBeNegative},
		{name: "NegativeResponseIDAccepted", opId: -1, expected: ""},
		{name: "LargeNegativeResponseIDAccepted", opId: -9999, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI}}
			assert.Equal(t, tt.expected, newSequencingTestServer().CallSequenceOperation(session, mioty.CmdStatusResponse, tt.opId))
		})
	}
}

// TestSCOperationIDStrictDecrement verifies that a response may name any
// issued service-center operation but never one not issued yet.
func TestSCOperationIDStrictDecrement(t *testing.T) {
	tests := []struct {
		name       string
		lastScOpId int64
		newOpId    int64
		expected   string
	}{
		{name: "NewestOperationAnswered", lastScOpId: -10, newOpId: -10, expected: ""},
		{name: "OlderPendingOperationAnswered", lastScOpId: -5, newOpId: -3, expected: ""},
		{name: "FirstOperationAnswered", lastScOpId: -1, newOpId: -1, expected: ""},
		{name: "UnissuedOperationRejected", lastScOpId: -1, newOpId: -2, expected: errOperationIDIncreasing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI, LastScOpId: tt.lastScOpId}}
			assert.Equal(t, tt.expected, newSequencingTestServer().CallSequenceOperation(session, mioty.CmdStatusResponse, tt.newOpId))
			assert.Equal(t, tt.lastScOpId, session.LastScOpId, "a response never moves the service-center counter")
		})
	}
}

// TestBSOperationIDMustBePositive verifies BSSCI §3.2-01: the base station
// starts its operations with positive IDs; 0 belongs to connect.
func TestBSOperationIDMustBePositive(t *testing.T) {
	tests := []struct {
		name     string
		opId     int64
		expected string
	}{
		{name: "PositiveIDAccepted", opId: 1, expected: ""},
		{name: "ZeroIDRejected", opId: 0, expected: errOperationIDNotPositive},
		{name: "NegativeIDRejected", opId: -1, expected: errOperationIDNotPositive},
		{name: "LargePositiveIDAccepted", opId: 99999, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI}}
			assert.Equal(t, tt.expected, newSequencingTestServer().CallSequenceOperation(session, mioty.CmdPing, tt.opId))
		})
	}
}

// TestBSOperationIDStrictIncrement verifies BSSCI §3.2-01: a new operation
// needs an ID above every earlier one; an open operation may be initiated
// again with its own ID.
func TestBSOperationIDStrictIncrement(t *testing.T) {
	tests := []struct {
		name         string
		lastBsOpId   int64
		openOpIDs    []int64
		newOpId      int64
		expected     string
		expectedLast int64
	}{
		{name: "NewOperationAdvancesCounter", lastBsOpId: 5, newOpId: 6, expected: "", expectedLast: 6},
		{name: "LowerIDRejected", lastBsOpId: 10, newOpId: 8, expected: errOperationIDBackwards, expectedLast: 10},
		{name: "CompletedIDReuseRejected", lastBsOpId: 20, newOpId: 20, expected: errOperationIDBackwards, expectedLast: 20},
		{name: "OpenOperationReinitiated", lastBsOpId: 20, openOpIDs: []int64{20}, newOpId: 20, expected: "", expectedLast: 20},
		{name: "FirstOperationAccepted", lastBsOpId: 0, newOpId: 1, expected: "", expectedLast: 1},
		{name: "GapAccepted", lastBsOpId: 100, newOpId: 200, expected: "", expectedLast: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI, LastBsOpId: tt.lastBsOpId}}
			session.restoreOpenBaseStationOperations(tt.openOpIDs...)
			assert.Equal(t, tt.expected, newSequencingTestServer().CallSequenceOperation(session, mioty.CmdPing, tt.newOpId))
			assert.Equal(t, tt.expectedLast, session.LastBsOpId)
		})
	}
}

// TestBSOperationCompletionNamesOpenOperation verifies that a base station
// completes an older operation after starting a newer one, and that a
// completion names an operation that is open.
func TestBSOperationCompletionNamesOpenOperation(t *testing.T) {
	server := newSequencingTestServer()
	session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI}}

	assert.Empty(t, server.CallSequenceOperation(session, mioty.CmdPing, 2))
	assert.Empty(t, server.CallSequenceOperation(session, mioty.CmdPing, 3))
	assert.Empty(t, server.CallSequenceOperation(session, mioty.CmdPingComplete, 2), "an older open operation completes")
	assert.Equal(t, errOperationNotOpen, server.CallSequenceOperation(session, mioty.CmdPingComplete, 2), "a completed operation is closed")
	assert.Equal(t, errOperationNotOpen, server.CallSequenceOperation(session, mioty.CmdAttachComplete, 7), "a never-started operation is not open")
	assert.Equal(t, errOperationIDNotPositive, server.CallSequenceOperation(session, mioty.CmdPingComplete, -3))

	assert.Empty(t, server.CallSequenceOperation(session, mioty.CmdErrorAck, 3), "an error exchange is not sequenced")
	assert.Equal(t, errOperationNotOpen, server.CallSequenceOperation(session, mioty.CmdPingComplete, 3), "the error exchange ended the operation")
}

// TestUnsequencedCommandsPassThrough verifies that connect messages and
// service-center-only commands are left to their own checks.
func TestUnsequencedCommandsPassThrough(t *testing.T) {
	server := newSequencingTestServer()
	session := &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: sequencingTestEUI}}
	for _, command := range []string{mioty.CmdConnect, mioty.CmdConnectComplete, mioty.CmdStatus, mioty.CmdAttachPropagate} {
		assert.Empty(t, server.CallSequenceOperation(session, command, 5), command)
	}
}
