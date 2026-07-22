# Changelog

All notable changes to KiloCenter are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases are cut by pushing a `vX.Y.Z` tag, which builds and publishes the
container images and creates the GitHub release.

## [1.3.0] - Unreleased

> **Migration required.** Endpoint and base-station EUIs are now stored as
> `BYTEA(8)` instead of a signed `BIGINT`. Run the database migrations before
> starting this version. On the old schema, EUIs above `INT64_MAX` (any EUI with
> the high bit set, such as `0xCAFECAFECAFECAFE`) overflowed silently.

### Added

- BSSCI protocol version negotiation. A base station may request a newer
  protocol version in `con`; the service center offers its own supported version
  in the `conRsp` `version` field, and the station either accepts it with
  `conCmp` or declines with `error` (BSSCI rev1 4.2, 5.3.2). Previously a
  station requesting `1.1.0` was rejected with error 71 and disconnected.
- Full unsigned EUI-64 support across the entire stack: wire decoding and
  encoding, persistence, gRPC and browser transport, certificates, and
  configuration. Values above `INT64_MAX` are no longer narrowed or truncated.
- Per-device blueprint snapshots with a System/Custom catalog, blueprint
  authoring in the web UI, and bulk version migration for deployed devices.
- Base-station availability and received-message time-series handlers.
- Certificate identity resolution for BSSCI connections: an EUI common name
  resolves against the registered station and an `org-<UUID>` common name
  resolves through the organization resolver, with SHA-256 fingerprint
  enforcement and CN/EUI binding in strict mode.
- `verify-export-tree` quality gate, which fails when key material, build
  output, or vendored dependency trees are tracked in the published tree.

### Changed

- **Breaking:** EUIs are stored as `BYTEA(8)` throughout the schema
  (see the migration note above).
- **Breaking:** `int64` fields that cross the browser boundary are annotated
  `[jstype = JS_STRING]` in the protobuf definitions, so values above 2^53
  survive JSON transport instead of losing precision.
- Each organization now receives its own tenant, and API keys are scoped to the
  organization targeted by the request rather than the caller's context.
- Downlink dispatch has a single owner of the `pending -> reserved -> queued`
  lifecycle, with exact-match reservation, idempotent confirmation, and
  `dlDataQueRsp` repair. `dlRxStatQry` pairs persist atomically.
- Session resume is strict: persisted operations are decoded and semantically
  reconstructed before `conRsp`. An operation that cannot be rebuilt rejects the
  resume with `EAGAIN` and preserves its record and queue state instead of
  discarding recoverable work.
- Service-center operation IDs follow a durable discipline: monotonic allocation
  is never rolled back, the counter is persisted before the pending record, and
  frames are written last. An ambiguous frame write closes the transport and
  recovers through resume with the original operation IDs.
- Event-stream polling no longer issues `COUNT(*)` on every request, and unary
  counts are cached with a TTL and request coalescing.
- Realtime cache invalidations in the web UI are coalesced, removing a refetch
  storm on busy tenants.
- Protocol servers consume narrow, purpose-built storage contracts instead of
  the shared storage facade.

### Fixed

- The gateway circuit breaker no longer counts a browser closing a realtime
  stream as an upstream failure, which previously opened the breaker and made
  the UI report "upstream circuit breaker is open". A half-open breaker now
  admits only slot-accounted unary probes, so recovery is decided by those
  probes alone.
- Attach and detach transitions are published to MQTT and surfaced in the
  activity feed.
- Uplinks are stamped with the serving tenant taken from the BSSCI session.
- Creating a base station with a duplicate EUI returns `AlreadyExists` instead
  of `Internal`.
- Tenant isolation: organization-scoped teardown, cross-tenant access for
  server administrators, and cascading tenant foreign keys.
- Base-station read queries and `endpoints.last_attached_bs_eui` handle BYTEA
  EUIs end-to-end.
- Session counter updates no longer reference a non-existent `last_message_at`
  column.
- ACK-only `dlDataQue` frames use the correct wire shape (`[[]]` rather than an
  empty outer array), which the Fraunhofer AVA base station rejected with
  error 22.
- MQTT events are never published for an unresolved organization.
- An incompletely wired server fails at startup rather than as a nil
  dereference under traffic, and a failed startup no longer leaves lifecycle
  state marked as started.

### Security

- Certificate issuance requires tenant context and verifies ownership before a
  certificate is produced.

## [1.2.0] - 2026-06-15

### Added

- `decoded_payload` is included in the uplink MQTT event.

### Fixed

- Downlink dispatch path corrected, and the downlink tab refreshes in realtime.

### Changed

- Generated protobuf files are excluded from static analysis.

## [1.1.2] - 2026-05-12

### Fixed

- 28 static-analysis findings resolved (code duplication and complexity),
  including extraction of `applyEndpointPostScan` and `EndpointSettingsSection`.

## [1.1.1] - 2026-05-12

### Fixed

- The login spinner no longer hangs when credentials are invalid.

## [1.1.0] - 2026-05-12

### Changed

- Large web-interface decomposition: the endpoints page, endpoint details,
  add/edit endpoint dialogs, downlink tab, base-station pages, and user detail
  view were split into focused components with pure validate/build helpers.
- BSSCI review cleanup across the protocol implementation.

### Fixed

- Migration 132 (seeding the default administrator) is properly idempotent.
- Self-contained ESLint configuration with lint and typecheck scripts, and
  TypeScript errors surfaced by `tsc -b` resolved.

## [1.0.0] - 2026-05-11

Initial public release of the KiloCenter MIOTY Service Center.

### Added

- Tag-triggered release workflow publishing semver-tagged container images.
- Project documentation: status badges, screenshot, and introduction.

[1.3.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.2.0...main
[1.2.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Kiloiot/kilo-service-center/releases/tag/v1.0.0
