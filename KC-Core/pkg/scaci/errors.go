// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

// POSIX Error Codes for SCACI protocol messages per MIOTY SCACI v1.0.0 §3.14
//
// These error codes are used in:
//   - Status responses (§3.5.2)
//   - Error messages (§3.14.1)
//
// Error codes follow standard POSIX errno values for cross-platform compatibility.
//
//revive:disable:var-naming
const (
	// POSIX_OK indicates successful operation (no error)
	POSIX_OK = 0

	// POSIX_EPERM indicates operation not permitted (permission denied for privileged operation)
	POSIX_EPERM = 1

	// POSIX_ENOENT indicates no such file or directory (requested resource does not exist)
	// In SCACI context: endpoint/session/operation not found
	POSIX_ENOENT = 2

	// POSIX_EIO indicates input/output error (generic I/O failure)
	// In SCACI context: database write failure, network I/O error
	POSIX_EIO = 5

	// POSIX_EAGAIN indicates resource temporarily unavailable (try again later)
	// In SCACI context: no suitable base station available for transmission
	POSIX_EAGAIN = 11

	// POSIX_ENOMEM indicates out of memory (system resource exhaustion)
	POSIX_ENOMEM = 12

	// POSIX_EINVAL indicates invalid argument (malformed request, invalid field values)
	// In SCACI context: missing required fields, invalid opId, type mismatch
	POSIX_EINVAL = 22

	// POSIX_ERANGE indicates numerical result out of range
	// In SCACI context: queue ID exceeds max int64, invalid numeric field value
	POSIX_ERANGE = 34

	// POSIX_EEXIST indicates file exists (resource already exists, duplicate entry)
	// In SCACI context: duplicate queue ID, conflicting endpoint registration
	POSIX_EEXIST = 17

	// POSIX_EPROTO indicates protocol error (protocol violation, unexpected message)
	// In SCACI context: wrong direction of operation, invalid handshake sequence
	POSIX_EPROTO = 71

	// POSIX_ENOTSUP indicates operation not supported (feature not implemented)
	// In SCACI context: unsupported SCACI command, unimplemented optional feature
	POSIX_ENOTSUP = 95

	// POSIX_ETIMEDOUT indicates connection timed out (operation timeout)
	// In SCACI context: operation timeout, no response within expected window
	POSIX_ETIMEDOUT = 110
)

//revive:enable:var-naming
