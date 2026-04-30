package bssci

// test_constants.go - Centralized test EUI constants
//
// All EUI values are ≤48-bit (0x0000FFFFFFFFFFFF max) to ensure JSON safety.
// JavaScript's Number type uses IEEE-754 float64 with a 53-bit mantissa,
// but only 48 bits preserve exact integer representation in JSON serialization
// per BSSCI/SCACI wire format requirements.
//
// Centralized test EUI values replace scattered literals across test files.

// Base Station EUI constants - used for session creation and BS-initiated operations
const (
	// TestBsEui01 - Primary test base station (default for most tests)
	TestBsEui01 uint64 = 0x0000112233445566

	// TestBsEui02 - Secondary base station for multi-BS scenarios
	TestBsEui02 uint64 = 0x0000AABBCCDDEE11

	// TestBsEui03 - Third base station for roaming tests
	TestBsEui03 uint64 = 0x0000998877665544

	// TestBsEui04 - Base station for session lookup tests
	TestBsEui04 uint64 = 0x0000123456789ABC

	// TestBsEuiMulti01 - First BS in concurrent session tests
	TestBsEuiMulti01 uint64 = 0x0000111111111111

	// TestBsEuiMulti02 - Second BS in concurrent session tests
	TestBsEuiMulti02 uint64 = 0x0000222222222222

	// TestBsEuiMulti03 - Third BS in concurrent session tests
	TestBsEuiMulti03 uint64 = 0x0000333333333333

	// TestBsEuiEdge - Base station for protocol edge case testing
	TestBsEuiEdge uint64 = 0x0000FFFFFFFFFFFF // Maximum 48-bit value
)

// Endpoint EUI constants - used for attach/detach/uplink/downlink operations
const (
	// TestEpEui01 - Primary test endpoint (default for most tests)
	TestEpEui01 uint64 = 0x0000112233445577

	// TestEpEui02 - Secondary endpoint for multi-EP scenarios
	TestEpEui02 uint64 = 0x0000223344556688

	// TestEpEui03 - Third endpoint for concurrent operations
	TestEpEui03 uint64 = 0x0000334455667799

	// TestEpEui04 - Fourth endpoint for bulk testing
	TestEpEui04 uint64 = 0x00004455667788AA

	// TestEpEuiDetach01 - Endpoint for detach operation tests
	TestEpEuiDetach01 uint64 = 0x0000010203040506

	// TestEpEuiDetach02 - Second endpoint for detach scenarios
	TestEpEuiDetach02 uint64 = 0x00000FEBABE11223

	// TestEpEuiDetach03 - Third endpoint for detach error cases
	TestEpEuiDetach03 uint64 = 0x0000111122223333

	// TestEpEuiDownlink01 - Endpoint for downlink queue tests
	TestEpEuiDownlink01 uint64 = 0x0000555566667777

	// TestEpEuiDownlink02 - Second endpoint for downlink scenarios
	TestEpEuiDownlink02 uint64 = 0x0000222233334444

	// TestEpEuiDlRx - Endpoint for DL RX status testing
	TestEpEuiDlRx uint64 = 0x000099AABBCCDDEE

	// TestEpEuiValidation - Endpoint for validation boundary tests
	TestEpEuiValidation uint64 = 0x0000AABBCCDDEEFF

	// TestEpEuiTenant01 - First endpoint for tenant isolation tests
	TestEpEuiTenant01 uint64 = 0x0000112233445566

	// TestEpEuiTenant02 - Second endpoint for tenant routing tests
	TestEpEuiTenant02 uint64 = 0x0000AABBCCDDEE22

	// TestEpEuiUnknown - Endpoint EUI not in database (for error cases)
	TestEpEuiUnknown uint64 = 0x0000DEADBEEF0000
)

// Service Center EUI constants - used for SCACI operations
const (
	// TestScEui01 - Primary test service center
	TestScEui01 uint64 = 0x0000001122334455

	// TestScEui02 - Secondary service center for roaming tests
	TestScEui02 uint64 = 0x0000009988776655

	// TestScEuiPropagation - Service center for propagate operation tests
	TestScEuiPropagation uint64 = 0x0000AABBCCDDEE00
)

// Special-purpose constants
const (
	// TestEuiSequential - Sequential pattern for ordering tests
	TestEuiSequential uint64 = 0x0000010203040506

	// TestEuiEncoding - Pattern with alternating bits for encoding tests
	TestEuiEncoding uint64 = 0x0000AAAA55550000

	// TestEuiZeroPadded - Minimal non-zero value for boundary tests
	TestEuiZeroPadded uint64 = 0x0000000000000001

	// TestEuiMax48Bit - Maximum 48-bit value for edge case testing
	TestEuiMax48Bit uint64 = 0x0000FFFFFFFFFFFF
)

// Compile-time assertions: Ensure all 28 EUI constants fit within 48 bits.
// These assertions verify each constant is ≤ 0x0000FFFFFFFFFFFF to ensure JSON safety
// (JavaScript Number type preserves exact integers up to 2^53, but only 48 bits are
// safe for BSSCI/SCACI wire formats per spec requirements).
//
// Issue #3-4 Fix B2: Replace simple right-shift with explicit value assertions.
// If any constant exceeds 48 bits, these const declarations will fail at compile time.
const (
	// Base Station EUI constants (8 constants)
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEui01      // Must be non-negative (within 48 bits)
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEui02      // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEui03      // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEui04      // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEuiMulti01 // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEuiMulti02 // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEuiMulti03 // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestBsEuiEdge    // Must be non-negative (max 48-bit value)

	// Endpoint EUI constants (13 constants)
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEui01         // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEui02         // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEui03         // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEui04         // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDetach01   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDetach02   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDetach03   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDownlink01 // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDownlink02 // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiDlRx       // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiValidation // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiTenant01   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiTenant02   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEpEuiUnknown    // Must be non-negative

	// Service Center EUI constants (3 constants)
	_ uint64 = 0x0000FFFFFFFFFFFF - TestScEui01          // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestScEui02          // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestScEuiPropagation // Must be non-negative

	// Special-purpose constants (4 constants)
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEuiSequential // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEuiEncoding   // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEuiZeroPadded // Must be non-negative
	_ uint64 = 0x0000FFFFFFFFFFFF - TestEuiMax48Bit   // Must be non-negative (equals max)

	// Total: 28 constants verified (8 BS + 13 EP + 3 SC + 4 special)
)
