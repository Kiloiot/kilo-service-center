package scaciservices

import "errors"

// Firmware version fixtures for nested SCACI connect metadata tests.
const (
	testFirmwareVersion        = "v1.2.3"
	testFirmwareVersionInitial = "v1.0"
	testFirmwareVersionUpdated = "v2.0"
)

// Failure fixtures injected into mocked dependencies.
var (
	errTestEntropyExhausted   = errors.New("entropy exhausted")
	errTestDBConnectionFailed = errors.New("database connection failed")
)
