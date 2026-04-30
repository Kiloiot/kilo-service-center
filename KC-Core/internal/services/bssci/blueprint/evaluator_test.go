package blueprint

import (
	"math"
	"testing"
)

func TestEvaluateFunc_BareCurrentValue(t *testing.T) {
	eval := NewExpressionEvaluator()

	// Spec syntax: bare $ means $value (MIOTY Application Layer Spec Table 15)
	result, err := eval.EvaluateFunc("$ / 10", float64(255), nil, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f, ok := result.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result)
	}
	if math.Abs(f-25.5) > 0.001 {
		t.Errorf("expected 25.5, got %f", f)
	}
}

func TestEvaluateFunc_LegacyDollarValue(t *testing.T) {
	eval := NewExpressionEvaluator()

	// Legacy syntax: $value
	result, err := eval.EvaluateFunc("$value / 10", float64(512), nil, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if math.Abs(f-51.2) > 0.001 {
		t.Errorf("expected 51.2, got %f", f)
	}
}

func TestEvaluateFunc_SpecFieldRef(t *testing.T) {
	eval := NewExpressionEvaluator()
	fields := map[string]interface{}{
		"temperature": float64(25.5),
	}

	// Spec syntax: $<name> refers to another field (Table 15)
	result, err := eval.EvaluateFunc("$temperature + 273.15", float64(0), fields, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if math.Abs(f-298.65) > 0.001 {
		t.Errorf("expected 298.65, got %f", f)
	}
}

func TestEvaluateFunc_SpecFieldRefCombined(t *testing.T) {
	eval := NewExpressionEvaluator()
	fields := map[string]interface{}{
		"offset": float64(10.0),
	}

	// Spec syntax: $<name> refers to another decoded field
	result, err := eval.EvaluateFunc("$value + $offset", float64(100), fields, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if math.Abs(f-110.0) > 0.001 {
		t.Errorf("expected 110.0, got %f", f)
	}
}

func TestEvaluateFunc_CalibrationRef(t *testing.T) {
	eval := NewExpressionEvaluator()
	calibration := map[string]interface{}{
		"factor": float64(1.5),
	}

	result, err := eval.EvaluateFunc("$value * $calibration.factor", float64(100), nil, calibration)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if math.Abs(f-150.0) > 0.001 {
		t.Errorf("expected 150.0, got %f", f)
	}
}

func TestEvaluateCondition_BareCurrentValue(t *testing.T) {
	eval := NewExpressionEvaluator()
	fields := map[string]interface{}{
		"type": float64(1),
	}

	// Spec syntax with $<name>
	result, err := eval.EvaluateCondition("$type == 1", fields, nil)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}
	if !result {
		t.Error("expected condition to be true")
	}
}

func TestEvaluateCondition_SpecFieldRef(t *testing.T) {
	eval := NewExpressionEvaluator()
	fields := map[string]interface{}{
		"enabled": float64(0),
	}

	// Spec syntax: $<name> refers to another decoded field
	result, err := eval.EvaluateCondition("$enabled == 1", fields, nil)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}
	if result {
		t.Error("expected condition to be false")
	}
}

func TestEvaluateCondition_Empty(t *testing.T) {
	eval := NewExpressionEvaluator()
	result, err := eval.EvaluateCondition("", nil, nil)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}
	if !result {
		t.Error("empty condition should return true")
	}
}

func TestEvaluateFunc_Empty(t *testing.T) {
	eval := NewExpressionEvaluator()
	result, err := eval.EvaluateFunc("", float64(42), nil, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if f != 42 {
		t.Errorf("empty func should return current value, got %f", f)
	}
}

func TestEvaluateFunc_BareCurrentValueInArithmetic(t *testing.T) {
	eval := NewExpressionEvaluator()

	// "$*2" should normalize to "$value*2"
	result, err := eval.EvaluateFunc("$ * 2", float64(50), nil, nil)
	if err != nil {
		t.Fatalf("EvaluateFunc failed: %v", err)
	}
	f := result.(float64)
	if math.Abs(f-100.0) > 0.001 {
		t.Errorf("expected 100.0, got %f", f)
	}
}

func TestMergeCalibration(t *testing.T) {
	blueprintCal := map[string]interface{}{
		"factor": 1.0,
		"offset": 0.5,
	}
	deviceCal := map[string]interface{}{
		"factor": 1.5,
	}

	merged := mergeCalibration(blueprintCal, deviceCal)
	if merged["factor"] != 1.5 {
		t.Errorf("device cal should override blueprint, got %v", merged["factor"])
	}
	if merged["offset"] != 0.5 {
		t.Errorf("blueprint default should be preserved, got %v", merged["offset"])
	}
}

func TestMergeCalibration_NilInputs(t *testing.T) {
	merged := mergeCalibration(nil, nil)
	if merged != nil {
		t.Error("expected nil for both nil inputs")
	}
}
