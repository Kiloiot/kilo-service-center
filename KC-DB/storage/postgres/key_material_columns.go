package postgres

import "github.com/Kiloiot/kilo-service-center/pkg/keycrypto"

// KeyMaterialColumn names a BYTEA column that stores key material as a binary
// keycrypto envelope.
type KeyMaterialColumn struct {
	Table      string
	Column     string
	PrimaryKey string
}

// keyMaterialColumns is the single list of binary key columns: the offline
// rekey command converts exactly these and keyMaterialGate inspects exactly
// these.
var keyMaterialColumns = []KeyMaterialColumn{
	{Table: "endpoints", Column: "nwk_key", PrimaryKey: "id"},
	{Table: "endpoints", Column: "app_key", PrimaryKey: "id"},
	{Table: "endpoint_sessions", Column: "session_key", PrimaryKey: "id"},
	{Table: "messages", Column: "nwk_sn_key", PrimaryKey: "id"},
	{Table: "messages_archive", Column: "nwk_sn_key", PrimaryKey: "id"},
}

// KeyMaterialColumns returns every BYTEA column that stores key material.
func KeyMaterialColumns() []KeyMaterialColumn {
	return append([]KeyMaterialColumn(nil), keyMaterialColumns...)
}

// Key material surfaces that are not a whole BYTEA column.
const (
	KeyMaterialSurfaceTLSKey             = "basestations.tls_key"
	KeyMaterialSurfacePendingMetadataKey = "bssci_pending_operations.metadata.encryptedKey"
	KeyMaterialSurfacePendingRecordKey   = "bssci_pending_operations.operation_data.nwkSnKey"
)

// KeyMaterialField is a stored key location that is not a whole BYTEA column.
// unconverted is the read-only probe keyMaterialGate runs for it.
type KeyMaterialField struct {
	Surface     string
	unconverted keyMaterialProbe
}

// keyMaterialFields is the single list of key fields: the offline rekey
// command binds a converter to each Surface and keyMaterialGate probes each
// one, so a key location added here is both converted and gated.
var keyMaterialFields = []KeyMaterialField{
	{
		Surface: KeyMaterialSurfaceTLSKey,
		unconverted: textEnvelopeProbe(
			`SELECT EXISTS (SELECT 1 FROM basestations WHERE tls_key <> '' AND NOT starts_with(tls_key, $1))`),
	},
	{
		Surface: KeyMaterialSurfacePendingMetadataKey,
		unconverted: textEnvelopeProbe(
			`SELECT EXISTS (SELECT 1 FROM bssci_pending_operations WHERE metadata ? 'encryptedKey' AND NOT starts_with(metadata ->> 'encryptedKey', $1))`),
	},
	{
		// A recovery record carrying its cleartext key blocks the upgrade by
		// its presence alone.
		Surface: KeyMaterialSurfacePendingRecordKey,
		unconverted: keyMaterialProbe{
			query: `SELECT EXISTS (SELECT 1 FROM bssci_pending_operations WHERE operation_data ? 'nwkSnKey')`,
		},
	},
}

// KeyMaterialFields returns every key location that is not a whole BYTEA column.
func KeyMaterialFields() []KeyMaterialField {
	return append([]KeyMaterialField(nil), keyMaterialFields...)
}

// textEnvelopeProbe binds the text envelope marker to $1 of query, which finds
// a stored text value that does not begin with it.
func textEnvelopeProbe(query string) keyMaterialProbe {
	return keyMaterialProbe{query: query, args: []interface{}{keycrypto.TextEnvelopePrefix()}}
}
