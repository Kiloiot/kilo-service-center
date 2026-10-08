package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestKeyMaterialGate_ProbesEveryDeclaredKeyLocation proves the gate reads
// the same list the rekey command converts: one probe per key column and key
// field, followed by the retired endpoint_keys tables.
func TestKeyMaterialGate_ProbesEveryDeclaredKeyLocation(t *testing.T) {
	var want []string
	for _, col := range KeyMaterialColumns() {
		want = append(want, col.Table+"."+col.Column)
	}
	for _, field := range KeyMaterialFields() {
		want = append(want, field.Surface)
	}
	for _, retired := range retiredKeyTableProbes {
		want = append(want, retired.surface)
	}

	probes := keyMaterialProbes()
	got := make([]string, 0, len(probes))
	for _, p := range probes {
		assert.NotEmpty(t, p.probe.query, "surface %s has no probe", p.surface)
		got = append(got, p.surface)
	}
	assert.Equal(t, want, got)
}
