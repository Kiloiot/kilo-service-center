package blueprint

import (
	"errors"
	"testing"
)

const (
	specValid = `{"version":"1.0","typeEui":"70B3D59CD0000095",
		"uplink":[{"id":0,"payload":[{"name":"t","size":8,"type":"uint"}]}]}`
	specNoUplinkNoDownlink = `{"version":"1.0","typeEui":"70B3D59CD0000095"}`
	specDuplicateFormatID  = `{"version":"1.0","typeEui":"70B3D59CD0000095",
		"uplink":[{"id":1,"payload":[{"name":"a","size":8,"type":"uint"}]},
		          {"id":1,"payload":[{"name":"b","size":8,"type":"uint"}]}]}`
	specUndefinedComponent = `{"version":"1.0","typeEui":"70B3D59CD0000095",
		"component":{"known":{"size":8,"type":"uint"}},
		"uplink":[{"id":0,"payload":[{"name":"t","component":"missing"}]}]}`
	specNotJSON = `{"version":`
)

func TestParseAndValidate_AcceptsAValidSpec(t *testing.T) {
	spec, err := ParseAndValidate([]byte(specValid))
	if err != nil {
		t.Fatalf("ParseAndValidate: %v", err)
	}
	if spec.Version != "1.0" || spec.TypeEUI != "70B3D59CD0000095" || len(spec.Uplink) != 1 {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}

func TestParseAndValidate_ParseFailuresAreParseErrors(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantToken string
	}{
		{name: "not json", spec: specNotJSON, wantToken: ErrInvalidBlueprintJSON},
		{name: "undefined component", spec: specUndefinedComponent, wantToken: ErrComponentRefNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, parseErr := ParseSpec([]byte(tt.spec))
			if parseErr == nil {
				t.Fatal("ParseSpec accepted the spec")
			}
			_, err := ParseAndValidate([]byte(tt.spec))
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("want *ParseError, got %T: %v", err, err)
			}
			if pe.Token != tt.wantToken {
				t.Fatalf("token = %q, want %q", pe.Token, tt.wantToken)
			}
			if err.Error() != parseErr.Error() {
				t.Fatalf("text = %q, want the parse text %q", err.Error(), parseErr.Error())
			}
		})
	}
}

func TestParseAndValidate_ValidationFailuresAreValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantToken string
	}{
		{name: "no uplink and no downlink", spec: specNoUplinkNoDownlink, wantToken: ErrMissingUplinkDefs},
		{name: "duplicate format id", spec: specDuplicateFormatID, wantToken: ErrDuplicateFormatID},
		{name: "no type eui", spec: `{"version":"1.0","uplink":[{"id":0,"payload":[]}]}`, wantToken: ErrMissingTypeEUI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAndValidate([]byte(tt.spec))
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *ValidationError, got %T: %v", err, err)
			}
			if ve.Token != tt.wantToken {
				t.Fatalf("token = %q, want %q", ve.Token, tt.wantToken)
			}
		})
	}
}
