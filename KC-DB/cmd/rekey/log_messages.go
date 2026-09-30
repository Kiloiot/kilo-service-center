package main

// Operator-facing report formats for the rekey command.
const (
	reportFmtHeader             = "rekey %s report\n"
	reportFmtSurfaceCounts      = "  %-45s envelope=%d plaintext=%d legacy=%d legacy_locked=%d malformed=%d converted=%d failed=%d\n"
	reportFmtEndpointKeysCounts = "  endpoint_keys: rows=%d archive_rows=%d resolved=%d exported=%d conflicts=%d\n"
	reportFmtEndpointKeysExport = "  endpoint_keys export written: %s (%d rows, master-key envelope)\n"
	reportFmtConflict           = "    conflict: %s\n"
	reportFmtConversionFailed   = "  %s id=%s digest=%s: conversion failed: %v\n"
)

// warnPublishedDevelopmentKey explains what -allow-published-development-key
// means for the rows it converts.
const warnPublishedDevelopmentKey = "WARNING: converting legacy rows sealed under the published development passphrase (KC_ENCRYPTION_KEY was never set). " +
	"Anyone with the repository could decrypt them, so the keys they hold were never secret; rotate any that must stay confidential."

// Fatal diagnostics printed to stderr before a nonzero exit.
const (
	fatalPrefix                 = "rekey: "
	fatalFmtUnknownMode         = "unknown mode %q"
	fatalMsgLegacyFlagsConflict = "-legacy-passphrase-env and -allow-published-development-key are mutually exclusive"
	fatalFmtEnvVarEmpty         = "environment variable %s is empty"
	fatalFmtOpenDatabase        = "open database: %v"
	fatalFmtConnectDatabase     = "connect to database: %v"
	fatalFmtReconcile           = "endpoint_keys reconciliation: %v"
	fatalFmtInspect             = "endpoint_keys inspection: %v"
	warnFmtCloseDatabase        = "close database: %v"
)

// Error vocabulary for conversion and reconciliation failures.
const (
	errFmtEndpointKeysRow        = "endpoint_keys id=%s: %w"
	errWrapCloseRows             = "close rows"
	errWrapRollback              = "roll back"
	errFmtInvalidDBPort          = "invalid DB_PORT: %w"
	errMsgUnresolvedArchiveRows  = "unresolved endpoint_keys/archive rows exist; pass -archive-export to write the encrypted export"
	errFmtDeleteExportedArchive  = "delete exported archive rows: %w"
	errFmtReadBackMismatchColumn = "read-back mismatch for endpoint %d %s"
	errFmtWriteExport            = "write export: %w"
	errFmtSurfaceConvert         = "%s: %w"
	errFmtNoSurfaceScanner       = "key surface %s has no scanner"
	errFmtScannerCount           = "%d surface scanners for %d declared key fields"
	errFmtRowsMatchedConcurrent  = "row changed concurrently (%d rows matched)"
	errFmtReadBackDecrypt        = "read-back decrypt: %w"
	errMsgReadBackMismatch       = "read-back mismatch"
	errFmtNoDestinationColumn    = "endpoint_keys id=%s key_type=%s has no destination column"
	errFmtKeyValueWrongLength    = "endpoint_keys id=%s key_value is not a %d-byte key"
	errFmtEndpointMissing        = "endpoint_keys id=%s references endpoint %d (tenant %d) which does not exist"
	errFmtEnvelopeNoDecrypt      = "endpoint_keys id=%s: endpoint %d %s envelope does not decrypt"
	errFmtDifferentKeyDigest     = "endpoint_keys id=%s: endpoint %d %s holds a different key (digest %s vs %s)"
	errFmtDifferentPreEnvelope   = "endpoint_keys id=%s: endpoint %d %s holds different pre-envelope bytes"
)
