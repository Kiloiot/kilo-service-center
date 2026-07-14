package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlueprintSnapshot_ToBlueprint validates snapshot->Blueprint synthesis: id, hex type_eui->bytes, spec passthrough.
func TestBlueprintSnapshot_ToBlueprint(t *testing.T) {
	src := uuid.New()
	snap := BlueprintSnapshot{
		SpecJSON:          json.RawMessage(`{"format":1}`),
		Version:           "1.2.3",
		TypeEUI:           "70b3d59cd0000094",
		SourceBlueprintID: src.String(),
		IsSystem:          false,
	}
	bp, err := snap.ToBlueprint()
	require.NoError(t, err)
	assert.Equal(t, src, bp.ID)
	assert.Equal(t, "1.2.3", bp.Version)
	assert.Equal(t, []byte{0x70, 0xb3, 0xd5, 0x9c, 0xd0, 0x00, 0x00, 0x94}, bp.TypeEUI)
	assert.JSONEq(t, `{"format":1}`, string(bp.SpecJSON))
}

// TestBlueprintSnapshot_ToBlueprint_InvalidID: a malformed source id is a hard error, not a silent bad decode.
func TestBlueprintSnapshot_ToBlueprint_InvalidID(t *testing.T) {
	snap := BlueprintSnapshot{SourceBlueprintID: "not-a-uuid", SpecJSON: json.RawMessage(`{}`)}
	_, err := snap.ToBlueprint()
	require.Error(t, err)
}

// TestBlueprintSnapshot_JSONRoundTrip: blueprint_snapshot bytes unmarshal back into the struct.
func TestBlueprintSnapshot_JSONRoundTrip(t *testing.T) {
	src := uuid.New()
	raw := []byte(`{"spec_json":{"a":1},"version":"1.0.0","type_eui":"70b3d59cd0000094","source_blueprint_id":"` + src.String() + `","is_system":true}`)
	var snap BlueprintSnapshot
	require.NoError(t, json.Unmarshal(raw, &snap))
	assert.True(t, snap.IsSystem)
	bp, err := snap.ToBlueprint()
	require.NoError(t, err)
	assert.Equal(t, src, bp.ID)
	assert.Len(t, bp.TypeEUI, 8)
	assert.JSONEq(t, `{"a":1}`, string(bp.SpecJSON))
}
