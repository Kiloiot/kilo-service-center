package encoding

// Error-wrap contexts and error format strings, grouped by the file that
// uses them; the shared group serves multiple files.
const (
	// data.go
	errWrapDecodeBase64 = "failed to decode base64"
)
