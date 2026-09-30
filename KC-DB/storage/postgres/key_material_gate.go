package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// ErrUnconvertedKeyMaterial reports key material that is not yet an
// authenticated envelope when an upgrade would move the schema past 000143.
// The offline rekey command converts it; the upgrade then continues.
var ErrUnconvertedKeyMaterial = errors.New("stored key material is not converted to authenticated envelopes; run the rekey command (-mode=apply, then -mode=verify) before migrating further")

// keyMaterialGateVersion is the last schema version a database may reach
// while key material is unconverted: 000143 is the first version whose
// endpoint key columns fit an envelope, and every later migration and the
// runtime readers assume each key already is one.
const keyMaterialGateVersion uint = 143

// keyMaterialProbe is a read-only query answering whether one surface still
// holds something the runtime cannot read.
type keyMaterialProbe struct {
	query string
	args  []interface{}
}

// surfaceProbe is the probe of one named surface.
type surfaceProbe struct {
	surface string
	probe   keyMaterialProbe
}

// binaryKeyProbeFmt finds a non-empty value of one key column that does not
// begin with the envelope marker bound to $1.
const binaryKeyProbeFmt = `SELECT EXISTS (SELECT 1 FROM %s WHERE length(%s) > 0 AND substring(%s FROM 1 FOR length($1::bytea)) <> $1::bytea)`

// retiredKeyTableProbes find rows in the retired endpoint_keys tables, which
// migration 000145 drops only when empty and the rekey command reconciles.
var retiredKeyTableProbes = []surfaceProbe{
	{surface: "endpoint_keys", probe: keyMaterialProbe{query: `SELECT EXISTS (SELECT 1 FROM endpoint_keys)`}},
	{surface: "endpoint_keys_archive", probe: keyMaterialProbe{query: `SELECT EXISTS (SELECT 1 FROM endpoint_keys_archive)`}},
}

// keyMaterialProbes lists the probe of every key column, every key field and
// every retired key table.
func keyMaterialProbes() []surfaceProbe {
	probes := make([]surfaceProbe, 0, len(keyMaterialColumns)+len(keyMaterialFields)+len(retiredKeyTableProbes))
	magic := keycrypto.EnvelopeMagic()
	for _, col := range keyMaterialColumns {
		probes = append(probes, surfaceProbe{
			surface: col.Table + "." + col.Column,
			probe: keyMaterialProbe{
				// #nosec G201 -- identifiers come from the fixed key column list.
				query: fmt.Sprintf(binaryKeyProbeFmt, col.Table, col.Column, col.Column),
				args:  []interface{}{magic},
			},
		})
	}
	for _, field := range keyMaterialFields {
		probes = append(probes, surfaceProbe{surface: field.Surface, probe: field.unconverted})
	}
	return append(probes, retiredKeyTableProbes...)
}

// keyMaterialGate holds the schema at keyMaterialGateVersion until every key
// surface the rekey command converts holds only envelopes and the retired
// endpoint_keys tables are empty. It only reads, so a refusal leaves the
// schema clean at that version.
type keyMaterialGate struct{}

func (keyMaterialGate) Version() uint { return keyMaterialGateVersion }

func (keyMaterialGate) Check(ctx context.Context, db *sql.DB) error {
	var unconverted []string
	for _, p := range keyMaterialProbes() {
		var found bool
		if err := db.QueryRowContext(ctx, p.probe.query, p.probe.args...).Scan(&found); err != nil {
			return fmt.Errorf(errFmtProbeKeyMaterial, p.surface, err)
		}
		if found {
			unconverted = append(unconverted, p.surface)
		}
	}
	if len(unconverted) == 0 {
		return nil
	}
	return fmt.Errorf(errFmtUnconvertedKeyMaterial, ErrUnconvertedKeyMaterial,
		keyMaterialGateVersion, strings.Join(unconverted, ", "))
}
