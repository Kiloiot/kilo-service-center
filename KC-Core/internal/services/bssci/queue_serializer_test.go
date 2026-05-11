package bssciservices

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

func TestBuildDLDataQueueComplete(t *testing.T) {
	serializer := NewQueueSerializer()
	opId := int64(12345)

	result := serializer.BuildDLDataQueueComplete(opId)

	// Verify command field
	if cmd, ok := result["command"].(string); !ok || cmd != "dlDataQueCmp" {
		t.Errorf("expected command='dlDataQueCmp', got %v", result["command"])
	}

	// Verify opId field
	if id, ok := result["opId"].(int64); !ok || id != opId {
		t.Errorf("expected opId=%d, got %v", opId, result["opId"])
	}

	// Verify map contains exactly 2 fields per BSSCI §5.12
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}
}

func TestBuildDLDataResultResponse(t *testing.T) {
	serializer := NewQueueSerializer()
	opId := int64(67890)
	dlResult := &mioty.DLDataResult{
		QueId:  uint64(111222),
		Result: mioty.ResultSent,
	}

	result := serializer.BuildDLDataResultResponse(opId, dlResult)

	// Verify command field (key name and value)
	if cmd, ok := result["command"].(string); !ok || cmd != "dlDataResRsp" {
		t.Errorf("expected command='dlDataResRsp', got %v", result["command"])
	}

	// Verify opId field (key name and value)
	if id, ok := result["opId"].(int64); !ok || id != opId {
		t.Errorf("expected opId=%d, got %v", opId, result["opId"])
	}

	// Verify map contains exactly 2 fields per BSSCI §5.14.2
	if len(result) != 2 {
		t.Errorf("expected 2 fields (command, opId), got %d", len(result))
	}
}

func TestBuildDLDataResultResponseWithFailure(t *testing.T) {
	serializer := NewQueueSerializer()
	opId := int64(-99999)
	dlResult := &mioty.DLDataResult{
		QueId:  uint64(333444),
		Result: mioty.ResultExpired,
	}

	result := serializer.BuildDLDataResultResponse(opId, dlResult)

	// Verify command field (key name and value)
	if cmd, ok := result["command"].(string); !ok || cmd != "dlDataResRsp" {
		t.Errorf("expected command='dlDataResRsp', got %v", result["command"])
	}

	// Verify opId field (key name and value)
	if id, ok := result["opId"].(int64); !ok || id != opId {
		t.Errorf("expected opId=%d, got %v", opId, result["opId"])
	}

	// Verify map contains exactly 2 fields per BSSCI §5.14.2 (result type does not affect response structure)
	if len(result) != 2 {
		t.Errorf("expected 2 fields (command, opId), got %d", len(result))
	}
}

func TestBuildDLDataResultComplete(t *testing.T) {
	serializer := NewQueueSerializer()
	opId := int64(-54321)

	result := serializer.BuildDLDataResultComplete(opId)

	// Verify command field
	if cmd, ok := result["command"].(string); !ok || cmd != "dlDataResCmp" {
		t.Errorf("expected command='dlDataResCmp', got %v", result["command"])
	}

	// Verify opId field
	if id, ok := result["opId"].(int64); !ok || id != opId {
		t.Errorf("expected opId=%d, got %v", opId, result["opId"])
	}

	// Verify map contains exactly 2 fields per BSSCI §5.14
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}
}
