// Package blueprint provides types for MIOTY Application Layer blueprint payload decoding.
// These types are used by the decoder service and match the blueprint JSON specification.
package blueprint

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Meta contains blueprint metadata per MIOTY Application Layer Spec Table 4, section 2.2.4.
type Meta struct {
	Name   string `json:"name,omitempty"`
	Vendor string `json:"vendor,omitempty"`
}

// ComponentDef defines a reusable component referenced by payload components.
// Per MIOTY Application Layer Spec Table 12, section 2.2.9.
// LittleEndian is NOT on ComponentDef — it is format-level only per section 2.2.10.
type ComponentDef struct {
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Unit   string `json:"unit,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
	Func   string `json:"func,omitempty"`
}

// VirtualValue defines a computed value derived from decoded fields.
// Per MIOTY Application Layer Spec Table 15.
type VirtualValue struct {
	Name      string `json:"name"`
	Func      string `json:"func"`
	Unit      string `json:"unit,omitempty"`
	Condition string `json:"condition,omitempty"`
}

// Spec represents the parsed blueprint JSON specification.
// This matches the MIOTY Application Layer Specification structure.
type Spec struct {
	Version     string                  `json:"version"`
	TypeEUI     string                  `json:"typeEui"` // Hex string representation of 8-byte EUI
	Meta        *Meta                   `json:"meta,omitempty"`
	Component   map[string]ComponentDef `json:"component,omitempty"`   // Reusable component definitions (Table 12, section 2.2.9)
	Calibration map[string]interface{}  `json:"calibration,omitempty"` // Blueprint-level calibration defaults (section 2.2.13)
	Uplink      []PayloadFormat         `json:"uplink,omitempty"`
	Downlink    []PayloadFormat         `json:"downlink,omitempty"`
	Metadata    map[string]string       `json:"metadata,omitempty"` // Legacy metadata map
}

// PayloadFormat defines a single uplink or downlink payload format.
// Each format has a unique format ID and defines how to decode the payload.
type PayloadFormat struct {
	FormatID     uint8              `json:"-"` // Populated by UnmarshalJSON from "id" or "formatId"
	Name         string             `json:"name,omitempty"`
	Description  string             `json:"description,omitempty"`
	Components   []PayloadComponent `json:"-"`                      // Populated by UnmarshalJSON from "payload" or "components"
	Crypto       string             `json:"crypto,omitempty"`       // Encryption scheme/code (accepts numeric or string on input)
	LittleEndian bool               `json:"littleEndian,omitempty"` // Format-level byte order (section 2.2.10)
	Virtual      []VirtualValue     `json:"virtual,omitempty"`      // Computed virtual values
}

// payloadFormatJSON is the raw JSON shape used for unmarshalling PayloadFormat.
type payloadFormatJSON struct {
	// Spec field name (Table 6)
	ID *uint8 `json:"id,omitempty"`

	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`

	// Spec field name (Table 6)
	Payload []PayloadComponent `json:"payload,omitempty"`

	Crypto       json.RawMessage `json:"crypto,omitempty"`
	LittleEndian bool            `json:"littleEndian,omitempty"`
	Virtual      []VirtualValue  `json:"virtual,omitempty"`
}

// parseCryptoValue accepts either a numeric or string "crypto" field.
// Manufacturer blueprints commonly use numeric codes (e.g. 0), while legacy
// payloads may use string values.
func parseCryptoValue(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var intVal int
	if err := json.Unmarshal(raw, &intVal); err == nil {
		return strconv.Itoa(intVal), nil
	}

	var strVal string
	if err := json.Unmarshal(raw, &strVal); err == nil {
		return strVal, nil
	}

	return "", fmt.Errorf("%s", ResolveErrorMessage(ErrInvalidCryptoFieldType))
}

func marshalCryptoValue(crypto string) (json.RawMessage, error) {
	if crypto == "" {
		return nil, nil
	}
	// Preserve numeric representation when possible.
	if _, err := strconv.Atoi(crypto); err == nil {
		return json.RawMessage(crypto), nil
	}
	b, err := json.Marshal(crypto)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// UnmarshalJSON implements custom unmarshalling for spec field names
// per MIOTY Application Layer Spec Table 6.
func (f *PayloadFormat) UnmarshalJSON(data []byte) error {
	var raw payloadFormatJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.ID != nil {
		f.FormatID = *raw.ID
	}

	f.Name = raw.Name
	f.Description = raw.Description
	f.Components = raw.Payload

	crypto, err := parseCryptoValue(raw.Crypto)
	if err != nil {
		return err
	}
	f.Crypto = crypto
	f.LittleEndian = raw.LittleEndian
	f.Virtual = raw.Virtual

	return nil
}

// MarshalJSON implements custom marshalling to emit spec-compliant field names.
func (f PayloadFormat) MarshalJSON() ([]byte, error) {
	cryptoRaw, err := marshalCryptoValue(f.Crypto)
	if err != nil {
		return nil, err
	}

	return json.Marshal(payloadFormatJSON{
		ID:           &f.FormatID,
		Name:         f.Name,
		Description:  f.Description,
		Payload:      f.Components,
		Crypto:       cryptoRaw,
		LittleEndian: f.LittleEndian,
		Virtual:      f.Virtual,
	})
}

// PayloadComponent defines a single field within a payload format.
// Components are decoded in order from the payload bytes.
type PayloadComponent struct {
	Name        string                 `json:"name"`
	Size        int                    `json:"size"`                  // Size in bits
	Type        string                 `json:"type"`                  // uint, int, float, bool, bytes, string, enum, binary
	Offset      *int                   `json:"offset,omitempty"`      // Bit offset (optional, defaults to sequential)
	Unit        string                 `json:"unit,omitempty"`        // Display unit (e.g., "°C", "%")
	Scale       float64                `json:"scale,omitempty"`       // Multiplication factor
	Bias        float64                `json:"bias,omitempty"`        // Addition offset after scaling
	Func        string                 `json:"func,omitempty"`        // Expression for computed values
	Condition   string                 `json:"condition,omitempty"`   // Conditional evaluation expression
	Calibration string                 `json:"calibration,omitempty"` // Reference to calibration data key
	Enum        map[string]interface{} `json:"enum,omitempty"`        // Enum value mapping
	Component   string                 `json:"component,omitempty"`   // Reference to component definition key (Table 10, section 2.2.7.2)
	Label       string                 `json:"label,omitempty"`       // Human-readable display label (Table 9)
	Hidden      bool                   `json:"hidden,omitempty"`      // Whether to hide from display (Table 13)
	Description string                 `json:"description,omitempty"` // Component-level description
}

// DecodeResult represents the result of decoding a payload using a blueprint.
type DecodeResult struct {
	// Success indicates whether decoding completed without errors.
	Success bool `json:"success"`

	// DecodedData contains the decoded field values keyed by component name.
	// Values are typed according to the component type (int64, float64, bool, string, []byte).
	DecodedData map[string]interface{} `json:"decodedData,omitempty"`

	// ErrorCode contains the error token if Success is false.
	// Use ResolveErrorMessage() to get the human-readable message.
	ErrorCode string `json:"errorCode,omitempty"`

	// ErrorDetail contains additional context about the error.
	ErrorDetail string `json:"errorDetail,omitempty"`

	// FormatID is the format ID that was used for decoding.
	FormatID uint8 `json:"formatId,omitempty"`

	// BlueprintVersion is the version of the blueprint used.
	BlueprintVersion string `json:"blueprintVersion,omitempty"`
}

// DecodeRequest contains the inputs for a decode operation.
type DecodeRequest struct {
	// UserData is the raw payload bytes to decode.
	UserData []byte `json:"userData"`

	// FormatID is the format identifier from the MIOTY message.
	// If nil, the decoder will attempt to use format ID 0.
	FormatID *uint8 `json:"formatId,omitempty"`

	// CalibrationData contains per-device calibration overrides.
	// Keys match $calibration.key references in blueprint func expressions.
	CalibrationData map[string]interface{} `json:"calibrationData,omitempty"`
}

// ResolveComponents merges ComponentDef properties into payload components that
// reference them via the Component field. Per MIOTY Application Layer Spec section 2.2.7.2,
// component-level values (Size, Type, Unit, Func, Hidden) are inherited from the definition
// unless explicitly overridden on the payload component.
func (s *Spec) ResolveComponents() error {
	if len(s.Component) == 0 {
		return nil
	}

	resolveList := func(formats []PayloadFormat) error {
		for fi := range formats {
			for ci := range formats[fi].Components {
				comp := &formats[fi].Components[ci]
				if comp.Component == "" {
					continue
				}
				def, ok := s.Component[comp.Component]
				if !ok {
					return &ValidationError{
						Token:  ErrComponentRefNotFound,
						Detail: fmt.Sprintf("component %q references undefined key %q", comp.Name, comp.Component),
					}
				}
				// Merge: component-level overrides definition
				if comp.Size == 0 {
					comp.Size = def.Size
				}
				if comp.Type == "" {
					comp.Type = def.Type
				}
				if comp.Unit == "" {
					comp.Unit = def.Unit
				}
				if comp.Func == "" {
					comp.Func = def.Func
				}
				if !comp.Hidden {
					comp.Hidden = def.Hidden
				}
			}
		}
		return nil
	}

	if err := resolveList(s.Uplink); err != nil {
		return err
	}
	return resolveList(s.Downlink)
}

// ParseSpec parses a JSON blueprint specification.
// After unmarshalling, it resolves component references.
// Returns an error with ErrInvalidBlueprintJSON token if parsing fails.
func ParseSpec(specJSON []byte) (*Spec, error) {
	var spec Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, &ParseError{
			Token:  ErrInvalidBlueprintJSON,
			Detail: err.Error(),
		}
	}

	if err := spec.ResolveComponents(); err != nil {
		return nil, err
	}

	return &spec, nil
}

// ParseError represents a blueprint parsing error with an error token.
type ParseError struct {
	Token  string
	Detail string
}

func (e *ParseError) Error() string {
	return ResolveErrorMessage(e.Token) + ": " + e.Detail
}

// ValidationError represents a blueprint validation error with an error token.
type ValidationError struct {
	Token  string
	Detail string
}

func (e *ValidationError) Error() string {
	return ResolveErrorMessage(e.Token) + ": " + e.Detail
}

// DecodeError represents a payload decoding error with an error token.
type DecodeError struct {
	Token     string
	Component string // Name of the component that caused the error
	Detail    string
}

func (e *DecodeError) Error() string {
	if e.Component != "" {
		return ResolveErrorMessage(e.Token) + " (component: " + e.Component + "): " + e.Detail
	}
	return ResolveErrorMessage(e.Token) + ": " + e.Detail
}

// NewDecodeResult creates a successful decode result.
func NewDecodeResult(data map[string]interface{}, formatID uint8, version string) *DecodeResult {
	return &DecodeResult{
		Success:          true,
		DecodedData:      data,
		FormatID:         formatID,
		BlueprintVersion: version,
	}
}

// NewDecodeError creates a failed decode result.
func NewDecodeError(errorCode, errorDetail string) *DecodeResult {
	return &DecodeResult{
		Success:     false,
		ErrorCode:   errorCode,
		ErrorDetail: errorDetail,
	}
}

// GetUplinkFormat finds an uplink format by format ID.
// Returns nil if the format is not found.
func (s *Spec) GetUplinkFormat(formatID uint8) *PayloadFormat {
	for i := range s.Uplink {
		if s.Uplink[i].FormatID == formatID {
			return &s.Uplink[i]
		}
	}
	return nil
}

// GetDownlinkFormat finds a downlink format by format ID.
// Returns nil if the format is not found.
func (s *Spec) GetDownlinkFormat(formatID uint8) *PayloadFormat {
	for i := range s.Downlink {
		if s.Downlink[i].FormatID == formatID {
			return &s.Downlink[i]
		}
	}
	return nil
}

// Validate checks if the blueprint specification is valid.
// Returns a ValidationError if validation fails.
func (s *Spec) Validate() error {
	if s.Version == "" {
		return &ValidationError{Token: ErrMissingVersion}
	}
	if s.TypeEUI == "" {
		return &ValidationError{Token: ErrMissingTypeEUI}
	}
	if len(s.Uplink) == 0 && len(s.Downlink) == 0 {
		return &ValidationError{Token: ErrMissingUplinkDefs}
	}

	// Check for duplicate format IDs in uplink
	seenUplink := make(map[uint8]bool)
	for _, format := range s.Uplink {
		if seenUplink[format.FormatID] {
			return &ValidationError{
				Token:  ErrDuplicateFormatID,
				Detail: "duplicate uplink format ID: " + string(rune(format.FormatID)),
			}
		}
		seenUplink[format.FormatID] = true
	}

	// Check for duplicate format IDs in downlink
	seenDownlink := make(map[uint8]bool)
	for _, format := range s.Downlink {
		if seenDownlink[format.FormatID] {
			return &ValidationError{
				Token:  ErrDuplicateFormatID,
				Detail: "duplicate downlink format ID",
			}
		}
		seenDownlink[format.FormatID] = true
	}

	return nil
}
