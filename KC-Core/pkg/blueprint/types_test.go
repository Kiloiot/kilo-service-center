package blueprint

import (
	"encoding/json"
	"testing"
)

func TestParseSpec_SpecFieldNamesBasic(t *testing.T) {
	// Spec format uses "id" and "payload" with meta object
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"meta": {"name": "MUNIA-M", "vendor": "BehrTech"},
		"uplink": [{
			"id": 0,
			"name": "standard",
			"payload": [
				{"name": "temperature", "size": 16, "type": "int", "func": "$value / 10", "unit": "°C"},
				{"name": "humidity", "size": 16, "type": "int", "func": "$value / 10", "unit": "%"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if spec.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", spec.Version)
	}
	if spec.TypeEUI != "70B3D59CD0000095" {
		t.Errorf("expected typeEui 70B3D59CD0000095, got %s", spec.TypeEUI)
	}

	if spec.Meta == nil {
		t.Fatal("expected Meta to be populated")
	}
	if spec.Meta.Name != "MUNIA-M" {
		t.Errorf("expected Meta.Name MUNIA-M, got %s", spec.Meta.Name)
	}
	if spec.Meta.Vendor != "BehrTech" {
		t.Errorf("expected Meta.Vendor BehrTech, got %s", spec.Meta.Vendor)
	}

	// Uplink format
	if len(spec.Uplink) != 1 {
		t.Fatalf("expected 1 uplink format, got %d", len(spec.Uplink))
	}
	f := spec.Uplink[0]
	if f.FormatID != 0 {
		t.Errorf("expected id 0, got %d", f.FormatID)
	}
	if len(f.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(f.Components))
	}
	if f.Components[0].Name != "temperature" {
		t.Errorf("expected component name temperature, got %s", f.Components[0].Name)
	}
}

func TestParseSpec_SpecFieldNames(t *testing.T) {
	// Spec format uses "id" and "payload"
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"meta": {"name": "MUNIA-M", "vendor": "BehrTech"},
		"uplink": [{
			"id": 0,
			"name": "standard",
			"littleEndian": true,
			"payload": [
				{"name": "temperature", "size": 16, "type": "int", "func": "$/10", "unit": "°C"},
				{"name": "humidity", "size": 16, "type": "int", "func": "$/10", "unit": "%"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if spec.Meta == nil || spec.Meta.Name != "MUNIA-M" {
		t.Errorf("expected Meta.Name MUNIA-M from explicit meta object")
	}

	f := spec.Uplink[0]
	if f.FormatID != 0 {
		t.Errorf("expected id 0, got %d", f.FormatID)
	}
	if !f.LittleEndian {
		t.Error("expected littleEndian=true")
	}
	if len(f.Components) != 2 {
		t.Fatalf("expected 2 components from payload field, got %d", len(f.Components))
	}
}

func TestParseSpec_CryptoNumeric(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"uplink": [{
			"id": 0,
			"crypto": 0,
			"payload": [
				{"name": "temperature", "size": 16, "type": "int", "func": "$/10"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if len(spec.Uplink) != 1 {
		t.Fatalf("expected 1 uplink format, got %d", len(spec.Uplink))
	}
	if spec.Uplink[0].Crypto != "0" {
		t.Errorf("expected crypto code 0, got %q", spec.Uplink[0].Crypto)
	}
}

func TestParseSpec_CryptoString(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"uplink": [{
			"id": 0,
			"crypto": "0",
			"payload": [
				{"name": "temperature", "size": 16, "type": "int", "func": "$/10"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if len(spec.Uplink) != 1 {
		t.Fatalf("expected 1 uplink format, got %d", len(spec.Uplink))
	}
	if spec.Uplink[0].Crypto != "0" {
		t.Errorf("expected crypto string 0, got %q", spec.Uplink[0].Crypto)
	}
}

func TestParseSpec_ComponentResolution(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"component": {
			"16bitTemperature": {"size": 16, "type": "int", "unit": "°C", "func": "$/10"},
			"16bitHumidity": {"size": 16, "type": "int", "unit": "%", "func": "$/10"}
		},
		"uplink": [{
			"id": 0,
			"payload": [
				{"name": "temperature", "component": "16bitTemperature"},
				{"name": "humidity", "component": "16bitHumidity"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	f := spec.Uplink[0]
	if len(f.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(f.Components))
	}

	temp := f.Components[0]
	if temp.Size != 16 {
		t.Errorf("expected resolved size 16, got %d", temp.Size)
	}
	if temp.Type != "int" {
		t.Errorf("expected resolved type int, got %s", temp.Type)
	}
	if temp.Unit != "°C" {
		t.Errorf("expected resolved unit °C, got %s", temp.Unit)
	}
	if temp.Func != "$/10" {
		t.Errorf("expected resolved func $/10, got %s", temp.Func)
	}

	hum := f.Components[1]
	if hum.Size != 16 {
		t.Errorf("expected resolved size 16, got %d", hum.Size)
	}
	if hum.Unit != "%" {
		t.Errorf("expected resolved unit %%, got %s", hum.Unit)
	}
}

func TestParseSpec_ComponentResolutionOverride(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"component": {
			"sensor": {"size": 16, "type": "int", "unit": "raw"}
		},
		"uplink": [{
			"id": 0,
			"payload": [
				{"name": "temperature", "component": "sensor", "unit": "°C", "func": "$/10"}
			]
		}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	temp := spec.Uplink[0].Components[0]
	// Unit should be overridden by component-level value
	if temp.Unit != "°C" {
		t.Errorf("expected component-level unit override °C, got %s", temp.Unit)
	}
	// Size should be inherited from definition
	if temp.Size != 16 {
		t.Errorf("expected inherited size 16, got %d", temp.Size)
	}
	// Func should be component-level override (definition has none)
	if temp.Func != "$/10" {
		t.Errorf("expected component-level func $/10, got %s", temp.Func)
	}
}

func TestParseSpec_MissingComponentRef(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"component": {
			"sensor": {"size": 16, "type": "int"}
		},
		"uplink": [{
			"id": 0,
			"payload": [
				{"name": "temperature", "component": "nonExistent"}
			]
		}]
	}`

	_, err := ParseSpec([]byte(specJSON))
	if err == nil {
		t.Fatal("expected error for missing component ref")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if ve.Token != ErrComponentRefNotFound {
		t.Errorf("expected token %s, got %s", ErrComponentRefNotFound, ve.Token)
	}
}

func TestParseSpec_MetaParsing(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"meta": {"name": "ExplicitName", "vendor": "ExplicitVendor"},
		"uplink": [{"id": 0, "payload": [{"name": "v", "size": 8, "type": "uint"}]}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if spec.Meta.Name != "ExplicitName" {
		t.Errorf("expected Meta.Name ExplicitName, got %s", spec.Meta.Name)
	}
	if spec.Meta.Vendor != "ExplicitVendor" {
		t.Errorf("expected Meta.Vendor ExplicitVendor, got %s", spec.Meta.Vendor)
	}
}

func TestParseSpec_CalibrationDefaults(t *testing.T) {
	specJSON := `{
		"version": "1.0",
		"typeEui": "70B3D59CD0000095",
		"calibration": {"factor": 1.5, "offset": 0.1},
		"uplink": [{"id": 0, "payload": [{"name": "v", "size": 8, "type": "uint"}]}]
	}`

	spec, err := ParseSpec([]byte(specJSON))
	if err != nil {
		t.Fatalf("ParseSpec failed: %v", err)
	}

	if spec.Calibration == nil {
		t.Fatal("expected calibration data")
	}
	if spec.Calibration["factor"] != 1.5 {
		t.Errorf("expected factor 1.5, got %v", spec.Calibration["factor"])
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	spec := &Spec{TypeEUI: "AAAA", Uplink: []PayloadFormat{{}}}
	err := spec.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	ve := err.(*ValidationError)
	if ve.Token != ErrMissingVersion {
		t.Errorf("expected %s, got %s", ErrMissingVersion, ve.Token)
	}
}

func TestValidate_MissingTypeEUI(t *testing.T) {
	spec := &Spec{Version: "1.0", Uplink: []PayloadFormat{{}}}
	err := spec.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	ve := err.(*ValidationError)
	if ve.Token != ErrMissingTypeEUI {
		t.Errorf("expected %s, got %s", ErrMissingTypeEUI, ve.Token)
	}
}

func TestPayloadFormat_MarshalRoundTrip(t *testing.T) {
	original := PayloadFormat{
		FormatID:     3,
		Name:         "test",
		LittleEndian: true,
		Components: []PayloadComponent{
			{Name: "a", Size: 8, Type: "uint"},
		},
		Virtual: []VirtualValue{
			{Name: "computed", Func: "$a * 2"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed PayloadFormat
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.FormatID != 3 {
		t.Errorf("expected formatId 3, got %d", parsed.FormatID)
	}
	if !parsed.LittleEndian {
		t.Error("expected littleEndian=true")
	}
	if len(parsed.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(parsed.Components))
	}
	if len(parsed.Virtual) != 1 {
		t.Fatalf("expected 1 virtual, got %d", len(parsed.Virtual))
	}
}

func TestPayloadComponent_ComponentRef(t *testing.T) {
	// Verify the Component field is parsed
	compJSON := `{"name": "temp", "component": "16bitTemp", "unit": "°C"}`
	var comp PayloadComponent
	if err := json.Unmarshal([]byte(compJSON), &comp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if comp.Component != "16bitTemp" {
		t.Errorf("expected component ref 16bitTemp, got %s", comp.Component)
	}
	if comp.Label != "" {
		t.Error("expected empty label")
	}
}

func TestPayloadComponent_NewFields(t *testing.T) {
	compJSON := `{"name": "temp", "size": 16, "type": "int", "label": "Temperature", "hidden": true, "description": "Ambient temperature"}`
	var comp PayloadComponent
	if err := json.Unmarshal([]byte(compJSON), &comp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if comp.Label != "Temperature" {
		t.Errorf("expected label Temperature, got %s", comp.Label)
	}
	if !comp.Hidden {
		t.Error("expected hidden=true")
	}
	if comp.Description != "Ambient temperature" {
		t.Errorf("expected description, got %s", comp.Description)
	}
}
