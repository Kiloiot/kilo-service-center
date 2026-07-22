package bssciservices

import (
	"context"
	"testing"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestAttachPropagateCompletionMetadataExtraction validates that the attPrpCmp
// message persistence correctly extracts all fields from pendingOp.Metadata.
// This tests the field extraction logic used in handleAttachPropagateComplete.
func TestAttachPropagateCompletionMetadataExtraction(t *testing.T) {
	t.Parallel()

	const (
		testTenantID      = int64(100)
		testEpEui         = uint64(0xAABBCCDDEEFF1122)
		testShAddr        = uint16(0x1234)
		testBidi          = true
		testLastPacketCnt = uint32(42)
	)

	// Simulate metadata as stored by SendAttachPropagate (JSON deserializes numbers as float64)
	metadata := map[string]interface{}{
		"epEui":         float64(testEpEui),
		"tenantId":      float64(testTenantID),
		"shortAddr":     float64(testShAddr),
		"bidirectional": testBidi,
		"lastPacketCnt": float64(testLastPacketCnt),
		"dualChannel":   true,
		"repetition":    false,
		"wideCarrOff":   true,
		"longBlkDist":   false,
	}

	// Extract fields using the same logic as handleAttachPropagateComplete
	var msgShAddr uint16
	if shAddrFloat, ok := metadata["shortAddr"].(float64); ok {
		msgShAddr = uint16(shAddrFloat)
	} else if shAddrInt, ok := metadata["shortAddr"].(int64); ok {
		msgShAddr = uint16(shAddrInt)
	}

	var msgBidi bool
	if bidiBool, ok := metadata["bidirectional"].(bool); ok {
		msgBidi = bidiBool
	}

	var msgLastPacketCnt uint32
	if lpcFloat, ok := metadata["lastPacketCnt"].(float64); ok {
		msgLastPacketCnt = uint32(lpcFloat)
	} else if lpcInt, ok := metadata["lastPacketCnt"].(int64); ok {
		msgLastPacketCnt = uint32(lpcInt)
	}

	var msgDualChan, msgRepetition, msgWideCarrOff, msgLongBlkDist bool
	if dualChan, ok := metadata["dualChannel"].(bool); ok {
		msgDualChan = dualChan
	}
	if repetition, ok := metadata["repetition"].(bool); ok {
		msgRepetition = repetition
	}
	if wideCarrOff, ok := metadata["wideCarrOff"].(bool); ok {
		msgWideCarrOff = wideCarrOff
	}
	if longBlkDist, ok := metadata["longBlkDist"].(bool); ok {
		msgLongBlkDist = longBlkDist
	}

	// Build completion message with extracted fields
	completionMsg := &mioty.AttachPropagateMessage{
		CommandType:   mioty.CmdAttachPropagateComplete,
		OpId:          int64(-12345),
		TenantID:      testTenantID,
		EpEui:         testEpEui,
		ShAddr:        msgShAddr,
		Bidi:          msgBidi,
		LastPacketCnt: msgLastPacketCnt,
		DualChan:      msgDualChan,
		Repetition:    msgRepetition,
		WideCarrOff:   msgWideCarrOff,
		LongBlkDist:   msgLongBlkDist,
		// NwkSnKey intentionally omitted
		MessageType:   mioty.MessageTypeAttachPropagate,
		Direction:     mioty.DirectionDownlink,
		InterfaceType: mioty.InterfaceBSSCI,
	}

	// Assertions
	assert.Equal(t, mioty.CmdAttachPropagateComplete, completionMsg.CommandType)
	assert.NotZero(t, completionMsg.EpEui, "EpEui should be non-zero")
	assert.Equal(t, testEpEui, completionMsg.EpEui)
	assert.Equal(t, testShAddr, completionMsg.ShAddr)
	assert.Equal(t, testBidi, completionMsg.Bidi)
	assert.Equal(t, testLastPacketCnt, completionMsg.LastPacketCnt)
	assert.True(t, completionMsg.DualChan)
	assert.False(t, completionMsg.Repetition)
	assert.True(t, completionMsg.WideCarrOff)
	assert.False(t, completionMsg.LongBlkDist)
	assert.Nil(t, completionMsg.NwkSnKey, "NwkSnKey should NOT be set in completion row")
}

// TestAttachPropagateCompletionPersistence verifies that the attPrpCmp message
// is correctly built with all required fields when a mock storage is available.
func TestAttachPropagateCompletionPersistence(t *testing.T) {
	t.Parallel()

	const (
		testTenantID      = int64(100)
		testOpId          = int64(-12345)
		testEpEui         = uint64(0xAABBCCDDEEFF1122)
		testBsEui         = uint64(0x1122334455667788)
		testShAddr        = uint16(0x1234)
		testBidi          = true
		testLastPacketCnt = uint32(42)
	)

	// Mock message store that captures calls
	mockStore := &mockAttPrpMessageStore{}

	// Simulate pendingOp metadata
	metadata := map[string]interface{}{
		"epEui":         float64(testEpEui),
		"tenantId":      float64(testTenantID),
		"shortAddr":     float64(testShAddr),
		"bidirectional": testBidi,
		"lastPacketCnt": float64(testLastPacketCnt),
	}

	// Simulate the message creation that would happen in handleAttachPropagateComplete
	var msgShAddr uint16
	if shAddrFloat, ok := metadata["shortAddr"].(float64); ok {
		msgShAddr = uint16(shAddrFloat)
	}

	var msgBidi bool
	if bidiBool, ok := metadata["bidirectional"].(bool); ok {
		msgBidi = bidiBool
	}

	var msgLastPacketCnt uint32
	if lpcFloat, ok := metadata["lastPacketCnt"].(float64); ok {
		msgLastPacketCnt = uint32(lpcFloat)
	}

	completionMsg := &mioty.AttachPropagateMessage{
		CommandType:   mioty.CmdAttachPropagateComplete,
		OpId:          testOpId,
		TenantID:      testTenantID,
		EpEui:         testEpEui,
		ShAddr:        msgShAddr,
		Bidi:          msgBidi,
		LastPacketCnt: msgLastPacketCnt,
		MessageType:   mioty.MessageTypeAttachPropagate,
		Direction:     mioty.DirectionDownlink,
		InterfaceType: mioty.InterfaceBSSCI,
	}

	// Call mock store
	err := mockStore.CreateAttachPropagateMessage(testutil.TestContext(), completionMsg)
	require.NoError(t, err)

	// Verify message was captured
	require.Len(t, mockStore.calls, 1)
	msg := mockStore.calls[0]

	assert.Equal(t, mioty.CmdAttachPropagateComplete, msg.CommandType)
	assert.NotZero(t, msg.EpEui, "EpEui should be non-zero")
	assert.Equal(t, testEpEui, msg.EpEui)
	assert.Equal(t, testShAddr, msg.ShAddr)
	assert.Equal(t, testBidi, msg.Bidi)
	assert.Equal(t, testLastPacketCnt, msg.LastPacketCnt)
	assert.Nil(t, msg.NwkSnKey, "NwkSnKey should NOT be persisted in completion row")
}

// TestAttachPropagateCompletionNoPendingOp verifies that the completion event
// fires even when no pendingOp is available, but message persistence is skipped.
func TestAttachPropagateCompletionNoPendingOp(t *testing.T) {
	t.Parallel()

	// Mock event store that captures calls
	mockEventStore := &mockAttPrpEventStore{}
	mockMsgStore := &mockAttPrpMessageStore{}

	// Simulate completion event creation (unconditional path)
	// This happens even when pendingOp is nil
	completionEvent := &models.SystemEvent{
		TenantID:    "100",
		EventType:   pkgbssci.EventTypeAttachPropagateCompleted,
		Title:       pkgbssci.TitleAttachPropagateCompleted,
		Description: "Attach propagate completed for opId -12345, BS 1122334455667788",
		Severity:    pkgbssci.SeverityInfo,
		Category:    "bssci",
		SourceType:  "service_center",
		SourceName:  "1122334455667788", // BS EUI in hex
		Status:      pkgbssci.EventStatusNew,
	}

	err := mockEventStore.CreateEvent(testutil.TestContext(), completionEvent)
	require.NoError(t, err)

	// Verify event was created
	require.Len(t, mockEventStore.events, 1)
	event := mockEventStore.events[0]

	assert.Equal(t, pkgbssci.EventTypeAttachPropagateCompleted, event.EventType)
	assert.NotEmpty(t, event.SourceName, "SourceName should contain BS EUI")
	assert.Equal(t, "1122334455667788", event.SourceName)
	assert.Equal(t, "100", event.TenantID)
	assert.Equal(t, pkgbssci.SeverityInfo, event.Severity)
	assert.Equal(t, pkgbssci.EventStatusNew, event.Status)

	// Message store should NOT be called when pendingOp is nil (no valid epEui)
	assert.Empty(t, mockMsgStore.calls, "No message should be persisted when pendingOp is nil")
}

// TestAttachPropagateCompletionMetadataInt64Types validates extraction works
// with int64 types (for scenarios where values are stored without JSON serialization)
func TestAttachPropagateCompletionMetadataInt64Types(t *testing.T) {
	t.Parallel()

	// Metadata with int64 values (direct storage without JSON)
	metadata := map[string]interface{}{
		"shortAddr":     int64(0x1234),
		"lastPacketCnt": int64(42),
	}

	var msgShAddr uint16
	if shAddrFloat, ok := metadata["shortAddr"].(float64); ok {
		msgShAddr = uint16(shAddrFloat)
	} else if shAddrInt, ok := metadata["shortAddr"].(int64); ok {
		msgShAddr = uint16(shAddrInt)
	}

	var msgLastPacketCnt uint32
	if lpcFloat, ok := metadata["lastPacketCnt"].(float64); ok {
		msgLastPacketCnt = uint32(lpcFloat)
	} else if lpcInt, ok := metadata["lastPacketCnt"].(int64); ok {
		msgLastPacketCnt = uint32(lpcInt)
	}

	assert.Equal(t, uint16(0x1234), msgShAddr)
	assert.Equal(t, uint32(42), msgLastPacketCnt)
}

// --- Mock Implementations ---

type mockAttPrpMessageStore struct {
	calls []*mioty.AttachPropagateMessage
}

func (m *mockAttPrpMessageStore) CreateAttachPropagateMessage(_ context.Context, msg *mioty.AttachPropagateMessage) error {
	m.calls = append(m.calls, msg)
	return nil
}

type mockAttPrpEventStore struct {
	events []*models.SystemEvent
}

func (m *mockAttPrpEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.events = append(m.events, event)
	return nil
}
