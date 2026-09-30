package models

// Error-wrap contexts and error format strings, grouped by the file that
// uses them; the shared group serves multiple files.
const (
	// blueprint.go
	errWrapDecodeSnapshotTypeEui  = "decode snapshot type_eui"
	errWrapParseSourceBlueprintID = "parse source_blueprint_id"

	// endpoint.go
	errFmtCannotScanIntoEUI  = "cannot scan %T into EUI"
	errFmtEUIMustBe8BytesGot = "EUI must be 8 bytes, got %d"
)
