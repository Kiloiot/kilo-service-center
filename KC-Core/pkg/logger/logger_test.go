package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestExpandFieldsWithField(t *testing.T) {
	input := []interface{}{"k", "v", Field{Key: "error", Value: "x"}}
	result := expandZapFields(input)
	expected := []interface{}{"k", "v", "error", "x"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("index %d: expected %v, got %v", i, v, result[i])
		}
	}
}

func TestToMapErrorValue(t *testing.T) {
	m := toMap("error", fmt.Errorf("something broke"))
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
		format: FormatJSON,
		output: &buf,
	}
	// Use logWithFields which is the real path for ErrorContext/log calls
	l.logWithFields(ErrorLevel, "test msg", toMap("error", fmt.Errorf("real error text")))

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

func TestExpandFieldsPassthrough(t *testing.T) {
	input := []interface{}{"key1", "val1", "key2", 42}
	result := expandZapFields(input)
	if len(result) != len(input) {
		t.Fatalf("expected %d elements, got %d", len(input), len(result))
	}
	for i, v := range input {
		if result[i] != v {
			t.Errorf("index %d: expected %v, got %v", i, v, result[i])
		}
	}
}
