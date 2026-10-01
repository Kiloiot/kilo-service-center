package models

// RadioMetricsUpdate contains radio metrics with optional field support.
// Nil pointers indicate "do not update this field" (preserve DB value).
type RadioMetricsUpdate struct {
	SNR        float64
	RSSI       float64
	EqSNR      *float64 // nil = preserve existing value
	RxTime     int64
	RxDuration *int64  // nil = preserve existing value
	Profile    *string // nil = preserve existing value
}
