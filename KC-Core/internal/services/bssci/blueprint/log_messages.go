package blueprint

// Log messages for blueprint resolution and payload decoding. Message text is
// stable so operators can grep the log stream by exact string.
const (
	LogBlueprintSpecParseFailed        = "Failed to parse blueprint specification"
	LogBlueprintValidationFailed       = "Blueprint validation failed"
	LogBlueprintFormatIDNotFound       = "Format ID not found in blueprint"
	LogBlueprintPayloadTooShort        = "Payload too short for format"
	LogBlueprintPayloadDecodeFailed    = "Payload decode failed"
	LogBlueprintPayloadDecoded         = "Payload decoded successfully"
	LogBlueprintNoTypeEUI              = "No Type EUI provided, skipping blueprint resolution"
	LogBlueprintFoundByTypeEUI         = "Found blueprint by Type EUI"
	LogBlueprintTypeEUILookupError     = "Error looking up blueprint by Type EUI"
	LogBlueprintNotFoundForTypeEUI     = "No blueprint found for Type EUI"
	LogBlueprintSnapshotParseFailed    = "failed to parse endpoint blueprint snapshot"
	LogBlueprintSnapshotInvalid        = "invalid endpoint blueprint snapshot"
	LogBlueprintResolvedFromSnapshot   = "Resolved blueprint from endpoint snapshot"
	LogBlueprintModelDefaultError      = "Error getting default blueprint for model"
	LogBlueprintResolvedFromModel      = "Resolved blueprint from device model"
	LogBlueprintModelNoDefault         = "Device model has no default blueprint"
	LogBlueprintNoTypeEUIOrModel       = "Endpoint has no Type EUI or device model"
	LogBlueprintCalibrationParseFailed = "Failed to parse endpoint calibration data"
)
