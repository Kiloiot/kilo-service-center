// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0.
//
// This package provides the SCACI protocol server, session management, operation handlers,
// and service layer components for MIOTY Application Center integration.
//
// Key components:
//   - Server: SCACI protocol server with mutual TLS support
//   - HandshakeService: Connect/version negotiation/session resumption
//   - EndpointService: Register/deregister endpoint operations
//   - ULService: Uplink transmit scheduling
//   - DLService: Downlink queue/revoke operations
//   - EPStatus broadcast/replay: SC-initiated endpoint status notifications with
//     resume-safe operation tracking and MessagePack/JSON assembly validation.
//
// Recent additions:
//   - EPStatus replay edge-case coverage (missing epEui/epStatus, invalid hex,
//     malformed nonce/subpackets) with zap observer log assertions to ensure
//     warnings/errors are emitted during replay.
//   - BroadcastEPStatus JSON round-trip tests validate Subpackets struct storage
//     survives persistence and can be reconstructed for resume/replay.
//
// Spec coverage: MIOTY SCACI v1.0.0 (57/57 sections).
package scaci
