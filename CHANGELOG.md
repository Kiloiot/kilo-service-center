# Changelog

All notable changes to KiloCenter are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-07-26

> **Before upgrading, run the database migrations.** Device and base-station
> EUIs are now stored in a format that covers the complete EUI-64 range.
> Existing EUIs are converted by the migration; no manual work is needed.

### Added

- **Base stations using BSSCI 1.1 can now connect.** When a station requests a
  newer protocol version than the service center supports, the service center
  offers its own version and the station accepts or declines it. Previously such
  a station was rejected outright and could not be used at all.
- **The full EUI-64 range is supported**, including EUIs that begin with a high
  byte such as `CA:FE:…`. Devices and base stations with those identifiers were
  previously rejected or stored incorrectly.
- **Device blueprints are versioned per device.** Each device keeps a snapshot of
  the blueprint it was provisioned with, so editing a blueprint in the catalog no
  longer changes how already-deployed devices decode their payloads. The catalog
  now separates built-in (System) blueprints from your own (Custom) ones,
  blueprints can be authored directly in the web interface, and many devices can
  be moved to a new blueprint version in a single operation.
- **Base-station availability and message-volume history**, so uptime and traffic
  can be reviewed over time instead of only as a current value.
- **Base stations can be identified by the EUI in their client certificate**, in
  addition to organization-based certificates.

### Changed

- Each organization now has its own tenant, and an API key grants access only to
  the organization it was issued for.
- Downlink delivery is more dependable: a queued downlink is claimed by exactly
  one delivery attempt, confirmations are safe to repeat, and a message is no
  longer lost or sent twice when a connection drops mid-transfer.
- A base station that reconnects after a network interruption resumes its
  previous session more safely. Downlinks and operations queued before the
  interruption are preserved and retried rather than silently dropped.
- Event lists and dashboards are noticeably faster on installations with large
  message volumes, and the web interface refreshes less aggressively.

### Fixed

- The web interface no longer reports "upstream circuit breaker is open" and
  stops loading data after a live page, such as a device or base-station detail
  view, has been left open for a while.
- Device attach and detach events now appear in the activity feed and are
  published to MQTT.
- Uplinks are attributed to the correct organization when a device is heard by a
  base station belonging to another organization.
- Creating a base station with an EUI that already exists reports a clear
  "already exists" message instead of a generic internal error.
- Data belonging to one organization can no longer surface in another, including
  while an organization is being removed.
- Acknowledgement-only downlinks, which carry no payload, are now accepted by
  Fraunhofer AVA base stations; they were previously rejected with error 22.
- Base-station and device pages no longer fail to load for certain EUIs.

### Security

- Certificates can only be issued for base stations in your own organization.
  Issuance now verifies ownership before a certificate is produced.

## [1.2.0] - 2026-06-15

### Added

- Decoded payload values are included in the MQTT uplink message, so
  integrations no longer have to decode payloads themselves.

### Fixed

- Downlinks are delivered reliably, and the downlink tab updates in real time.

## [1.1.2] - 2026-05-12

### Fixed

- Consistency and stability improvements across the device and base-station
  screens.

## [1.1.1] - 2026-05-12

### Fixed

- Signing in with invalid credentials no longer leaves the login button spinning
  indefinitely.

## [1.1.0] - 2026-05-12

### Changed

- The device, base-station and user screens were reorganized for faster loading
  and more consistent behavior.

### Fixed

- A fresh installation reliably creates the default administrator account.
- Several interface errors on the device and base-station screens.

## [1.0.0] - 2026-05-11

Initial public release of KiloCenter, the MIOTY Service Center: base-station and
device management, payload blueprints, downlinks, and MQTT integration.

### Added

- Container images are built and published automatically for every release.

[1.3.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Kiloiot/kilo-service-center/releases/tag/v1.0.0
