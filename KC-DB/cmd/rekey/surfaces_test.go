package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// testUndeclaredSurface names a key location postgres does not declare.
const testUndeclaredSurface = "undeclared_table.key"

// TestSurfaces_ConvertEveryDeclaredKeyLocation ties the command to the one
// list postgres declares: every key column and every key field is scanned,
// in declaration order, and nothing else is.
func TestSurfaces_ConvertEveryDeclaredKeyLocation(t *testing.T) {
	surfaces, err := (&rekeyer{}).surfaces()
	require.NoError(t, err)

	var want []string
	for _, col := range postgres.KeyMaterialColumns() {
		want = append(want, col.Table+"."+col.Column)
	}
	for _, field := range postgres.KeyMaterialFields() {
		want = append(want, field.Surface)
	}
	got := make([]string, 0, len(surfaces))
	for _, s := range surfaces {
		got = append(got, s.name)
	}
	assert.Equal(t, want, got)
}

func TestBindFieldScanners_RefusesADeclaredFieldWithoutAScanner(t *testing.T) {
	fields := append(postgres.KeyMaterialFields(), postgres.KeyMaterialField{Surface: testUndeclaredSurface})
	scanners := (&rekeyer{}).fieldScanners()
	scanners[testUndeclaredSurface+"-other"] = scanners[postgres.KeyMaterialSurfaceTLSKey]

	_, err := bindFieldScanners(fields, scanners)
	require.Error(t, err)
	assert.Contains(t, err.Error(), testUndeclaredSurface)
}

func TestBindFieldScanners_RefusesAScannerForAnUndeclaredField(t *testing.T) {
	scanners := (&rekeyer{}).fieldScanners()
	scanners[testUndeclaredSurface] = scanners[postgres.KeyMaterialSurfaceTLSKey]

	_, err := bindFieldScanners(postgres.KeyMaterialFields(), scanners)
	require.Error(t, err)
}
