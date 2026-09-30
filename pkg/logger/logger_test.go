package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestToMapOddFields(t *testing.T) {
	m := toMap("key1", "value1", "lonely")
	if m["key1"] != "value1" {
		t.Errorf("expected value1, got %v", m["key1"])
	}
	if _, ok := m["lonely"]; ok {
		t.Errorf("unexpected dangling key mapped")
	}
}

func TestToMapNonStringKey(t *testing.T) {
	m := toMap("k1", "v1", 123, "v2")
	if len(m) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m))
	}
	if m["k1"] != "v1" {
		t.Errorf("expected v1, got %v", m["k1"])
	}
}

func TestToMapField(t *testing.T) {
	m := toMap("key1", "value1", Field{Key: "error", Value: "something failed"})
	if m["key1"] != "value1" {
		t.Errorf("expected value1, got %v", m["key1"])
	}
	if m["error"] != "something failed" {
		t.Errorf("expected 'something failed', got %v", m["error"])
	}
}

// Error fixtures for the field-conversion tests.
var (
	errTestSomethingBroke = errors.New("something broke")
	errTestRealErrorText  = errors.New("real error text")
)

func TestToMapErrorValue(t *testing.T) {
	m := toMap("error", errTestSomethingBroke)
	val, ok := m["error"]
	if !ok {
		t.Fatal("expected 'error' key in map")
	}
	str, isString := val.(string)
	if !isString {
		t.Fatalf("expected string value, got %T", val)
	}
	if str != "something broke" {
		t.Errorf("expected 'something broke', got %q", str)
	}
}

func TestToMapErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	l := &logger{
		level:  ErrorLevel,
		format: formatJSON,
		output: &buf,
	}
	// Use logWithFields which is the real path for ErrorContext/log calls
	l.logWithFields(ErrorLevel, "test msg", toMap("error", errTestRealErrorText))

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}
	errVal, ok := parsed["error"]
	if !ok {
		t.Fatal("expected 'error' field in JSON output")
	}
	if errVal != "real error text" {
		t.Errorf("expected 'real error text', got %v", errVal)
	}
}
