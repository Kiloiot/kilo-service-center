package certificates

import "time"

// Timing policies for generated certificate artifacts.
const (
	// certDownloadWindow bounds how long generated certificate artifacts
	// remain downloadable before cleanup removes them.
	certDownloadWindow = 15 * time.Minute
	// hoursPerDay converts a remaining validity to whole days.
	hoursPerDay = 24
)

// operationEncryptKey labels the private-key encryption step in log fields.
const operationEncryptKey = "encrypt"
