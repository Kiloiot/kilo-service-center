# Changelog

All notable changes to KiloCenter are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-09-30

KiloCenter 2.0 is a major release. Upgrading from 1.x takes a few manual steps, described in
[Key material migration](GitBook/06-Operations/02-key-material-migration.md).

**Why this is 2.0 and not 1.4:**

- **New settings are required before the upgrade.**
  - Set a master key (`KILOCENTER_MASTER_KEY`). It encrypts your endpoint and base station keys
    at rest, and the services do not start without it.
  - When KC-Gateway, KC-Core and KC-Identity run on separate hosts or containers, also set a
    shared internal secret (`KILOCENTER_INTERNAL_AUTH_PEER_SECRET`).
  - Upgrade KC-Core and KC-Identity together.
- **Your stored keys are converted.** The upgrade re-encrypts every stored key under the master
  key, and after that the database can no longer be used with a 1.x release. Back it up first.
- **Parts of the API changed.**
  - The organization quota fields are removed.
  - Endpoint keys are masked on every read.
  - Every call now checks the user's role and answers `PERMISSION_DENIED` when the role does not
    cover it.
  - The web interface's WebSocket connection is replaced by gRPC-web server streams.

**Cleanup and bug fixes.** A major cleanup release:
- Dead code, unused settings and unused exported declarations are removed, along with the old
  System screen and the WebSocket transport.
- Large components are split into smaller, focused parts.
- Hard-coded values are moved into shared catalogs and configuration.
- More than fifty bugs are fixed across uplinks, downlinks, sessions, certificates and the web
  interface. They are listed under Fixed below.

**New in the web interface:**
- **Live updates.** Dashboards, lists and activity refresh within about a second of a message
  arriving, with no manual refresh.
- **Activity on every base station and endpoint.** One table shows each uplink with its radio
  metrics, payload and decoded values inline. Rows expand to the receiving stations, their
  subpackets and the ulData message.
- **Traffic.** Uplinks show decoded values when the endpoint has a blueprint. Downlinks can be
  queued, revoked and followed to delivery.
- **Blueprints.** A blueprint can be assigned from the endpoint's page.
- **Logs.** An error center and an event log with filters replace the old activity feed, and
  the dashboard shows an Alerts card.
- **Base stations.** A map of the stations, plus certificate status and expiry. A station's
  certificate bundle can be downloaded, and the server certificate renewed without losing the
  names stations connect by.
- **Users and roles.** Administrators grant roles. A refused password shows the reason, and API
  keys can be created in the community edition.

**New in the service center:**
- **Protocol conformance.** Uplinks, downlinks, attach and detach follow BSSCI v1.0.0 and SCACI
  v1.0.0 end to end.
- **Sessions.** Base station and Application Center sessions survive a restart and resume.
- **Uplinks.** One uplink received by several base stations is delivered once, listing every
  receiving station.
- **Downlinks.** Downlinks are queued, revoked, expired and reported to the Application Center
  that queued them.
- **Tenant isolation and security.**
  - Roles are enforced on every call, and secrets are audited before they are handed out.
  - Base station certificates are bound to their station.
  - SCACI sessions, queue ids and results stay within their organization.
  - Calls between the services are authenticated.
- **New API fields.** The endpoint's latest RSSI, SNR and equivalent SNR, the downlink accept
  and acknowledge times, the base station certificate expiry, and a password policy.
- **New MQTT fields and events.** `dlOpen`, `responseExp` and `dlAck` on uplinks, delivery details
  on downlink results, more `command/down` options, and `event/downlink_queued`.

> **Breaking:** organization quotas are gone. `can_have_base_stations`,
> `max_base_station_count` and `max_endpoint_count` no longer exist on
> `CreateOrganizationRequest`, `UpdateOrganizationRequest` or `Organization`
> (the field numbers and names are reserved), the UI no longer shows or edits
> them, and migration 000154 drops the columns. They were published but never
> enforced; a deployment that needs entitlements enforces them in the platform
> above the service center. Clients that still send the fields get them
> ignored as unknown fields; clients that read them must drop the code.

> **The user role switches are enforced, and existing accounts keep their
> access.** Every API call except sign-in, registration and the public release
> and onboarding calls needs a signed-in user whose roles cover it (Admin,
> Tenant Manager, Base Station Manager, Endpoint Manager); other calls return
> `PERMISSION_DENIED`. Migration 000176 grandfathers every account and active
> organization membership that exists when it runs: they gain Base Station
> Manager and Endpoint Manager, which reproduces the unrestricted data access
> they had, members who could manage members keep Tenant Manager, and nobody
> becomes an administrator. The migration records what it changed, so
> migrating down restores the previous switches. Accounts created after the
> upgrade, whether by self-registration, invitation or a first OpenID Connect
> sign-in, start without roles until an administrator grants one.
> Service-account API keys act as Base Station Manager and Endpoint Manager in
> their own organization only.
>
> **Upgrade note:** KC-Core no longer starts without `identity.address`, a stack
> run with authentication disabled no longer serves data calls, and the internal
> `IdentityInternalService.GetUserMembership` RPC is replaced by `GetUserRoles`,
> so KC-Core and KC-Identity must be upgraded together.

### Changed
- Endpoint keys are masked on every read. `GetEndPoint`, `ListEndPoints` and
  the `CreateEndPoint` and `UpdateEndPoint` responses return `nwk_sn_key` and
  `app_key` empty, with `nwk_sn_key_set` and `app_key_set` saying whether each
  is stored. An Endpoint Manager or Admin reads a key by calling `GetEndPoint`
  with `reveal_keys` (`ENDPOINT_KEY_NWK_SN_KEY`, `ENDPOINT_KEY_APP_KEY`), and
  every reveal writes an `endpoint.keys_revealed` audit event naming who
  revealed which key of which endpoint, never the key. Clients that read keys
  from endpoint records must ask for them explicitly. In the web interface the
  Configuration tab shows each key as set or not set, and the edit dialog
  shows a stored key masked and empty, with a Reveal button that fetches it
  and a Copy button that works once it is revealed or a new key is entered.
  Leaving a key field empty keeps the stored key. A stored application key is
  removed with the field's Remove action, confirmed first and applied on save
  (`UpdateEndPoint` with `app_key` in the mask and no value); the network
  session key is mandatory, so it can be replaced but never removed. Every
  removal of a stored key writes an `endpoint.keys_removed` audit event naming
  who removed which key of which endpoint, never the key.
- The web interface has no System screen. The Application Center control
  plane, statistics and analytics, integrations, capabilities and the
  diagnostics bundle are API calls for the software that connects to the
  service center, each with its own role
  (`GitBook/05-Security/04-users-and-roles.md`); the dashboard shows Endpoint
  Managers the Application Center status and Base Station Managers the
  certificate expiry of their stations.
- A base station's and an endpoint's **Activity** tab is one table in the
  Events Log columns (Time, Event type, opId, Scope, Severity, Summary)
  instead of a terminal-style timeline. Events read as in the Events Log and
  expand to their data; uplinks read as `ulData` with their `opId`, the
  endpoint (on a station) or the receiving stations (on an endpoint) and
  their radio metrics, and expand to the receptions, the `ulData` message and
  the blueprint-decoded payload, which the Traffic tables now show too. Paging
  only offers pages the feed's cursor has reached, so a page number no longer
  shows another page's rows. The separate device **Logs** tab is gone; an
  endpoint's attach, detach or delete failure shows on every tab.
- The dashboard has no Platform Activity feed. The **Alerts** card reads like
  the Events Log and carries the live-connection indicator and a link to the
  Events Log, where the full platform history is.
- `ListBaseStationMessages` takes the uplink filters `ListMessages` has
  (`duplicate`, `dl_open`, `profile`, `mode`), and `BaseStationMessage`
  carries `dl_open`, `res_exp`, `dl_ack`, `op_id` and `format`, so a base
  station's Traffic tab lists the station's uplinks with the same columns and
  filters as the Traffic section. A base station manager sees them there.
- The Helm value `kcIdentity.config.auth.registrationEnabled` is removed. No
  template read it, so KC-Identity always ran with self-registration off, which
  is unchanged.
- A refused password is answered with its reason: `CreateUser`,
  `UpdateUserPassword` and `ChangePassword` return InvalidArgument with the
  password policy for a weak password (was Internal "failed to ...") and
  `ChangePassword` returns InvalidArgument "current password is incorrect"
  for a wrong current password.
- API keys can be created in the community edition (`CreateApiKey` returned
  "service not configured" there).
- The base station uplink export (`ExportBaseStationMessages`) carries the
  SCACI §3.8.1 ulData fields in both formats under the same camelCase names:
  `id`, `opId`, `epEui`, `bsEui`, `rxTime`, `rxDuration`, `packetCnt`, `snr`,
  `rssi`, `eqSnr`, `profile`, `mode`, `format`, `dlOpen`, `responseExp`,
  `dlAck`, `duplicate`, `userData`, `subpackets`. EUIs are hex, `opId` is a
  decimal string, `rxTime` is RFC 3339 in UTC with nanoseconds and `userData`
  is hex; before, the JSON export wrote EUIs, `opId` and `rxTime` as numbers
  that JavaScript rounded, and `userData` as base64, and the CSV lacked most
  fields and used snake_case headers.
- Every 64-bit identifier and nanosecond timestamp in the gRPC API is
  delivered to JavaScript clients as a string (`jstype = JS_STRING`): downlink
  `que_id`, BSSCI/SCACI `op_id` fields, `rx_time`/`tx_time`, the SCACI session
  `last_op_id_*` counters and `ReleaseInfo.sc_eui`. Generated JavaScript
  getters for these fields now return strings; Go, Python and other clients are
  unaffected.
- `ListScaciSessionsRequest` gains `can_resume_filter` (explicit presence) so a
  client can ask for non-resumable sessions; the implicit `can_resume` field is
  deprecated and still means "resumable only" when set.
- Uplink duplicates are recognized by the base station's reception time
  (`rxTime`) instead of the time they reach the service center, so a ulData a
  base station resends after a session resume merges into the original message.
- A failed uplink delivery to an Application Center or MQTT is retried until it
  succeeds, with an exponential backoff capped by the new
  `protocol.delivery.max_backoff` (default 5m); only deliveries that can never
  succeed (unknown or unconfigured channel, message gone) are parked.
- An over-the-air attach restarts the endpoint's packet counter, and the stored
  counter otherwise never moves backwards.
- The configured `protocol.roaming.cache_*` and `enable_audit_trail` settings
  now take effect, and every reception is judged against the tenant of the base
  station that received it.
- An upgrade from a release that stored plaintext key material stops at
  migration 000143 until the `rekey` command has converted every key: both
  `migrate up` and service start-up refuse to continue and leave the schema
  clean at 143. The Helm chart's pre-upgrade hook already runs `rekey` there.
- The Helm `rekey` hook reads its own hook-scoped Secret, decrypts rows sealed
  with the retired published development passphrase by default
  (`rekey.allowPublishedDevelopmentKey: true`, since the chart never set
  `KC_ENCRYPTION_KEY`), and can export retired `endpoint_keys` rows to a
  persistent volume (`rekey.archiveExport.existingClaim`).
  `rekey.legacyPassphraseSecretKey` is replaced by `rekey.legacyPassphrase`.
- SCACI session resume follows §3.3.1: `snAcOpId` is the lowest Application
  Center operation the service center must know and `snScOpId` the highest
  service center operation the Application Center has seen; a resumed session
  accepts the Application Center's reissued operations with their original ids
  and replays the service center's own uncompleted ones.
- A new SCACI session of an Application Center supersedes its earlier session
  and closes the older connection; a session only becomes usable after
  `conCmp`, and the TLS handshake plus connect must finish within
  `protocol.connection_establishment_timeout`.
- Every service center operation sent over SCACI is recorded for resume before
  it is written; a broadcast reports each failed session and continues with the
  rest.
- Every BSSCI and SCACI frame write is bounded by the new
  `protocol.socket_write_timeout` (default 10 s), so one stalled peer no longer
  blocks delivery to the others.
- A base station may complete its own operations out of order: a completion for
  an operation that is still open is accepted, and only new operations must use
  a higher id. Stations that sent their next uplink before completing the
  previous one were disconnected on every such overlap.
- BSSCI sessions survive a service center restart or crash and can be resumed.
- `/health/ready` reports only what KC-Core needs to serve its API (PostgreSQL
  and the internal gRPC server); `/health` still lists every component, and the
  BSSCI/SCACI rows no longer open connections to the listeners.
- KC-Core and KC-Identity trust gateway identity headers only from a caller that
  presents `internal_auth.peer_secret` when one is configured (set the same value
  on KC-Gateway, KC-Core and KC-Identity; empty keeps the previous behavior).
- The service center assigns every downlink queue id; an Application Center's own
  `queId` is kept per organization while its downlink is in flight and returned
  in its `dlDataRes`, so two tenants, or two organizations of a tenant, can use
  the same id.
- A downlink that is persisted is acknowledged to the Application Center even if
  the immediate hand-off to the base station fails; it is delivered later.
- `RevokeDownlink` revokes a pending downlink immediately (status `revoked`) and
  answers NotFound unless the queue id belongs to the requested endpoint and
  organization. `ListDownlinkQueue` lists the request organization only.
- A downlink the base station refuses is marked failed and reported as
  `invalid` to the Application Center and on MQTT `event/downlink_result`.
- A ulData with a malformed `subpackets` object, or with more than 200 bytes of
  `userData` (the radio payload limit), is answered with a BSSCI protocol error
  (EPROTO) and not stored. An `att` or `det` with a malformed `subpackets`
  object is refused the same way (BSSCI §3.10.1) instead of being accepted
  without it.
- When several base stations receive the same uplink, the endpoint's downlink
  window is filled with one queued downlink, not one per station.
- SCACI ulData reports `duplicate: true` when an endpoint reuses a packet counter
  it already sent outside the duplicate window; the uplink is still stored and
  delivered. Migration 000163 adds `messages.packet_cnt_reused`.
- SCACI `baseStations[].dlRxSnr`/`dlRxRssi` ride only the first uplink after the
  endpoint's DL RX status report instead of every later uplink.
- New downlink queue ids are at most 2^53-1, so JavaScript and other JSON clients
  read them exactly.
- MQTT `command/down` payloads are limited to 200 decoded bytes, the radio
  downlink limit, instead of 256 KB.
- Over-the-air attachment follows the radio specification: the attach signature
  is checked over the 16-byte initialization vector with a 4-byte attach counter
  (it used 15 bytes, so a real endpoint's over-the-air attach was refused), other
  base stations receive the session key instead of the pre-shared key, an
  attach response omits `shAddr` when the station assigned one, and concurrent
  identical attaches are served once.
- After an over-the-air detach the other base stations are told to drop the
  endpoint, and Application Centers receive `epStat` for attachments and
  detachments made through the service center as well.
- SCACI byte fields travel as arrays of numbers in both directions, as the
  specification defines them; binary values are still accepted. Out-of-range
  numbers are refused with code 34 instead of being truncated, a `dlDataQue`
  without `cntDepend` or `userData` is refused, `prio` is accepted in any numeric
  width, and JSON-framed Application Centers are served (replies stay
  MessagePack).
- `ListScaciQueues` lists the request organization only.
- A downlink queued ahead of time goes to the base station that received the
  endpoint's latest uplink with the best SNR, a roaming station of another
  tenant included, or, before the endpoint has been heard, to the station it
  was attached or propagated through. When that station is offline or cannot
  transmit, the downlink waits for the endpoint's next downlink window instead
  of going to a station that never hears the endpoint. A downlink for an
  endpoint no station has heard or attached yet waits for the window of its
  first `dlOpen` uplink too, instead of going to an arbitrary bidirectional
  station of its tenant. `QueryDLRXStatus` asks the same station.
- A downlink for an endpoint registered without `bidi` is refused: SCACI
  answers the `dlDataQue` with an error (code 95) and `SendDownlink` returns
  FailedPrecondition (KC-GRPC-ERR-280). No `dlDataQue` is sent to a base station
  whose session is not bidirectional.
- When several base stations report the same telegram, its downlink window goes
  to the first bidirectional one, so a receive-only station that reported it
  first no longer leaves the window unused. Migration 000165 adds
  `messages.dl_window_claimed`.
- When a base station starts a new session instead of resuming, the downlinks
  it held are queued again: a new session discards the station's state.
- A downlink waits at most `protocol.downlink_expiry.lifetime` (default 24h)
  for a downlink window. After that it is marked expired and reported as
  `expired` to the Application Center that queued it, on MQTT
  `event/downlink_result` and in `GetDownlinkResults`.
  `protocol.downlink_expiry.sweep_interval` (default 5s) and `batch_size`
  (default 100) tune the sweep; Helm `kcCore.config.protocol.downlinkExpiry`.
  A downlink a base station still holds expires the same way, and the base
  station is asked to drop it with `dlDataRev`. Its confirmation, and a
  `dlDataRes` for a downlink that already expired or was revoked, leave the
  reported outcome unchanged and are not reported again.
- SCACI `dlDataRev` revokes every downlink scheduled for the packet counter and
  nothing else: a downlink queued without packet counters is no longer revoked
  by whatever counter the Application Center names. With nothing scheduled for
  the counter, the revoke completes instead of answering with an error.
- `dlDataRes` reaches only the Application Center that queued the downlink
  (the one of the downlink's organization whose `dlDataQue` queued it), not
  every Application Center of the tenant; results of downlinks queued through
  gRPC or MQTT stay on those channels. The queue row stores that Application
  Center (migration 000184 adds `downlink_queue.ac_eui` and fills it for
  existing downlinks from their `dlDataQue` records), so a result still
  reaches it when the operation log could not record the `dlDataQue`. A
  downlink queued before the upgrade whose records name no Application Center,
  or several, is reported to none; the upgrade preflight lists them.
- While an Application Center is disconnected but may still resume its SCACI
  session, the service center records the `ulData`, `epStat` and `dlDataRes` it
  would have sent under the session's next operation ids. On resume they are
  reissued after `conCmp`, in operation id order and before any new operation,
  with ids continuing from before the connection loss; a new session discards
  them, and a resume of a session that is no longer held is refused. Held
  sessions survive a service center restart. The new
  `protocol.scaci_resume_max_pending_operations` (default 10000; Helm
  `kcCore.config.scaci.resumeMaxPendingOperations`) bounds what a session holds:
  the operation that would exceed it ends the session's resumability.
- An Application Center's `queId` may be any non-zero 64-bit value; values at or
  above 2^63 were refused with code 34, and `dlDataRes` returns the exact id.
  Migration 000166 changes `downlink_queue.ac_que_id` to NUMERIC(20,0).
- `GetDownlinkResults` and `ListDownlinkQueue` report the result `revoked` for a
  revoked downlink instead of an empty result.
- Shutdown waits for the downlink results still being forwarded to Application
  Centers.
- An endpoint's attachment status is the service center's decision.
  `AttachEndPoint`, `DetachEndPoint`, `UpdateEndPoint` with `status`,
  `CreateEndPoint` with `pre_attach`, a SCACI `reg` with `preAttach`, a SCACI
  `dereg`, and an over-the-air attach or detach set it, and only the call that
  changes it sends `epStat` to the owner's Application Centers, once: a
  `dereg` of an endpoint already detached announces nothing. A base station's
  attPrp or detPrp completion only records that station's confirmation: it no
  longer sets the status, sends `epStat` or publishes MQTT, so a late
  completion cannot undo a newer decision. An endpoint attached while no base
  station is connected is sent to each station when it connects.
- `CreateEndPoint` ignores the submitted endpoint's `status`: it is detached
  unless `pre_attach` attaches it. A pre-attached endpoint is stored together
  with its attachment, or not at all: one whose network key cannot be sent to
  the base stations is refused (KC-GRPC-ERR-10B) and stores nothing, so the
  same create can be sent again.
- A base station that connects or reconnects is sent attPrp only for the
  endpoints that are attached, not for every endpoint of its tenant. A base
  station that resumes its session kept the endpoints it held, so it is also
  sent detPrp for every endpoint whose detachment was decided after its
  connection was lost; one that starts a new session is not. The time of each
  decision is recorded in the new `attachment_changed_at` column (added as
  `status_changed_at` by migration 000168, backfilled from the last
  over-the-air detach of detached endpoints, and renamed by migration 000170),
  so a late base station report no longer moves it, and `last_detach_time`
  keeps only the reception time of the last over-the-air detach.
- `UpdateEndPoint` with `status` in `update_mask` attaches (`attached`) or
  detaches (`detached`) the endpoint through `AttachEndPoint` or
  `DetachEndPoint`, so the base stations are sent the change and Application
  Centers receive one `epStat`; the status was written directly before. An
  unchanged status changes nothing, `attaching` is refused with
  KC-GRPC-ERR-315 ("status must be attached or detached"). No other endpoint
  update changes the attachment status.
- `AttachEndPoint` and `DetachEndPoint` of an endpoint the tenant does not
  have answer not found (KC-GRPC-ERR-101) instead of a failed attach or
  detach.
- `CreateEndPoint` with an `ep_class` other than `A` or `Z` is refused with
  KC-GRPC-ERR-314 instead of failing in storage, and every endpoint is stored
  with the class its `bidi` flag implies, whatever class the writer passed.
- MQTT `event/attach` and `event/detach` are published once for every
  attachment and detachment the service center decides, over the air or made
  in KC-Web, through the API or by an Application Center; only an over-the-air
  one carries `bsEui`. An over-the-air attach is recorded when the base
  station completes it (`attCmp`).
- Every SCACI request without one of its mandatory fields is refused with
  `scaci.error.missing_mandatory_field` (code 22). This replaces the per-field
  checks, including `scaci.error.missing_packet_cnt`, which no longer exists:
  a `dlDataRev` without `packetCnt` now gets `missing_mandatory_field`.
- SCACI requests are decoded once. A JSON frame's mandatory fields are found
  by the same key matching as its other fields, and a frame with more than one
  defect is answered with the first defect the decoder meets; a frame with a
  single missing or invalid field is refused as before.
- A base station's frames no longer start one database write each: the
  session's operation counters are written by one writer at a time, which
  picks up the newest counters, and a slow write no longer holds back the next.
- Negative values in SCACI 64-bit unsigned fields (`epEui`, `bsEui`, `queId`,
  `acEui`) are refused with `scaci.error.field_out_of_range` (code 34) instead
  of wrapping into a different identifier.
- A SCACI `dereg` revokes the endpoint's downlinks through the revoke path, so a
  downlink a base station already holds is revoked there with `dlDataRev`
  instead of only in the store.
- `GetScaciStatus` requires an organization and counts the pending downlink
  operations of that organization only.
- The Helm chart rolls KC-Core and Mosquitto out with the `Recreate` strategy.
  A rolling update started the new KC-Core beside the old one, and its
  start-up marked the old pod's live BSSCI sessions disconnected; both pods
  also mount a `ReadWriteOnce` volume that attaches to one node at a time.
- A configuration with an empty `certificates.certgen_path`, `certs_dir` or
  `temp_dir`, or a `server_validity_days` or `cleanup_interval_min` of zero or
  less, is refused at start-up instead of silently replaced by the default;
  leave a setting out to get its default.
- `protocol.strict_org_resolution=true` with a SCACI listener requires
  `protocol.scaci_cert_tenant_mapping=true` in every edition, checked when
  KC-Core loads its configuration.
- KC-Identity caches the default organization of a tenant for
  `general.org_cache_ttl_minutes`, as KC-Core does, instead of querying the
  database on every request; both binaries share one organization resolver.

### Added
- MQTT `event/up` carries the whole uplink: `epEui`, `eqSnr`, `rxDuration`,
  `format`, `profile`, `mode`, `subpackets`, every receiving base station in
  `baseStations` (with `dlRxSnr`/`dlRxRssi` when reported), `duplicate` and
  `packetCntReused`, and the decode outcome in `decodeStatus`,
  `decodeErrorCode` and `blueprintTypeEui`. The earlier fields are unchanged.
- `AuthSettings.password_policy` (field 11, new message `PasswordPolicy`): the
  minimum and maximum length and the letter and digit requirements every new
  password must meet, so clients can state the rules before a password is
  refused.
- `user.password_changed` audit events for a user's own password change and an
  admin's reset, filed under the platform tenant with the actor as the event's
  user and the user whose password changed as `userId` in its data.
- `user.registered` audit events for self-registration, filed under the
  platform tenant with the new user as the event's user and as `userId` in its
  data; no credential enters the event.
- `EndPoint.last_rssi`, `last_snr`, `last_eq_snr` (fields 28-30): the latest
  reception of the endpoint, absent until it is heard; `EndPoint.serving_bs_eui`
  (field 31, `GetEndPoint` only): the base station a downlink queued now would
  go to. The endpoint detail shows them with its MIOTY configuration (short
  address in hex, class, counters, type EUI, carrier offset, radio flags and
  whether each key is set).
- `DownlinkMessage.accepted_at` (field 29): when the base station in `bs_eui`
  accepted the downlink with dlDataQueRsp (BSSCI §3.12); unset until a station
  accepts it and cleared when the downlink returns to pending. The endpoint's
  downlink queue shows the holding station and its acceptance, and both
  downlink tables list every payload with the packet counter it is valid for.
- `BaseStationMessage.dl_open`, `dl_ack` and `res_exp` (fields 23-25): the
  uplink flags of a base station's activity, as `Message` already carries them
  for an endpoint's; the web activity shows them with the endpoint, eqSNR and
  the payload in hex.
- `BaseStation.certificate_expires_at` (field 24, optional): the expiry of the
  certificate the service center issued for the station, on both the list and
  the detail, so the base station list shows it.
- `GetServerCertificateStatusResponse.renewal_names` (field 3): the names a
  server certificate issued now carries, subject first. The Certificates page
  lists them before a renewal is confirmed.
- `certificates.server_names`: further DNS names and IP addresses the server
  certificate carries; certgen takes them through `-san`.
- `DownlinkMessage.endpoint_acked_at` (field 28): when an uplink with `dlAck`
  acknowledged the downlink sent in the endpoint's previous window; the downlink
  results in the web interface show it as "Acknowledged by device". Migration
  000164 adds `downlink_queue.endpoint_acked_at`.
- `Message.op_id` (field 24): the BSSCI ulData operation id of the first
  reception, for joining an uplink to its event-log rows.
- `EndPoint.reattach_pending` (field 27, read-only): an edit changed an
  attached endpoint's attach propagate parameters (BSSCI §3.8.1) after the
  last attach propagate a base station completed, so the stations still hold
  the earlier profile until the endpoint is attached again. The web interface
  marks such an endpoint "Configuration pending re-attach". Migration 000173
  adds `endpoints.profile_changed_at`.
- MQTT `event/up` carries `dlOpen`, `responseExp` and `dlAck`.
- MQTT `event/downlink_result` carries `bsEui`, `txTime` and `packetCnt` when
  `result` is `sent`.
- MQTT `command/down` accepts `prio`, `format`, `responsePrio`, `dlWindReq`,
  `expOnly`, `dlRxStatQry`, counter-dependent `entries`
  (`[{"packetCnt":N,"data":"base64"}]`) and a correlation `ref`; an empty `data`
  queues a pure acknowledgement. Existing `{"data","confirmed"}` commands queue
  the same downlink as before.
- MQTT `event/downlink_queued` (the queue id of an accepted command) and
  `event/downlink_rejected` (a stable `code` and `message`), both echoing `ref`.
- MQTT `event/downlink_result` carries the `ref` of the `command/down` that
  queued the downlink on every result, so your application can match results
  to its commands without keeping its own list of queue ids. A `ref` longer
  than 128 bytes is refused with `mqtt.command.ref_too_long`. Migration 000187
  adds `downlink_queue.ref`.
- MQTT `event/downlink_result` with `"result": "acknowledged"`: published once
  when the endpoint confirms, with the acknowledgement flag of its next uplink,
  that it received a sent downlink. It carries the downlink's `queId`, your
  `ref` and the acknowledged window in `packetCnt`. When the endpoint's packet
  counter restarted, the downlink sent last in that window is the one
  acknowledged. Application Centers keep reading the flag from `ulData`, as
  SCACI defines no such result.

### Removed
- Organization and tenant quota fields, columns and UI (`can_have_base_stations`,
  `max_base_station_count`, `max_endpoint_count`, `tenants.max_basestations`,
  `tenants.max_endpoints`).
- `protocol.delivery.max_attempts` (Helm `delivery.maxAttempts`); old
  configuration files that still set it load unchanged.
- Unused exported Go declarations: `storage.ErrStorageNotAvailable` (KC-DB),
  the log fields `FieldHandshakeComplete`, `FieldMaxAllowed`,
  `FieldOperationID` and `FieldTotalSessionsCamel` (`pkg/logger` and the
  KC-Core `logger` package), `bssci.LogBSSCIUnsupportedOperationTypeStartEvent`
  and the KC-Core internal `ErrNilBackgroundWork`.
- gRPC-web over WebSocket, and with it the `grpc.web.enable_websockets` setting
  (`config.DefaultGRPCWebEnableWebsockets`, `GRPCWebConfig.EnableWebsockets`,
  log field `FieldWebsockets`). Every RPC is unary or server-streaming, so
  gRPC-web clients use the default HTTP transport, as KC-Web does. KC-Gateway
  had stopped routing WebSocket upgrades when it began stripping hop-by-hop
  headers; KC-Core's optional gRPC-web listener (`grpc.web.enabled`) now
  serves HTTP only too. Configuration files that still set
  `enable_websockets` load unchanged; the key is ignored.

### Security
- A secret is handed out only once its audit record is written. An endpoint
  key reveal (`GetEndPoint` with `reveal_keys`), a base station private key
  download (`DownloadBaseStationCertificate` or `DownloadCertificate` with
  `cert_type=key`) and a new API key (`CreateApiKey`) answer `INTERNAL`
  without the secret when the event store cannot take the record, instead of
  handing it out unrecorded. A private key refused this way stays stored and
  can be downloaded again once the event store is back; an API key refused
  this way is stored only as a hash, so nobody can use it. Every private key
  download now writes a `certificate.private_key_downloaded` audit event
  naming who downloaded the key of which station, never the key. A second
  download of a key that another download is taking at that moment is refused
  at once instead of waiting.
- Roles are enforced in the service center and reflected in KC-Web. A user
  holding only the endpoint manager role could register a base station and
  receive its private key, and a self-registered user without any role could
  list the tenant's endpoints, read their network session keys and see
  security and system events. Each RPC now declares the role it requires
  (`GitBook/05-Security/04-users-and-roles.md` lists them); endpoint keys go
  only to endpoint managers, base station certificates and keys only to base
  station managers, and security, system, audit and error events, alerts and
  the diagnostics bundle only to administrators. Event lists show only the
  categories a user's roles cover. A role change applies within
  `grpc.rbac_role_cache_ttl_seconds` without a new sign-in. A deactivated
  administrator no longer passes the server-admin check. `GetProfile` returns
  the caller's effective roles (`UserProfile.roles`).
- A base station's private key leaves the service center once, as the
  certificate dialog already promised: `DownloadBaseStationCertificate` with
  `cert_type=key` returned the decrypted stored key on every call, and
  `DownloadCertificate` served a bundle's key file to every download until the
  bundle was cleaned up. Both paths now take the stored key under a row lock,
  so exactly one download, even among concurrent ones, receives it; every later
  one is refused, and the bundle's key file is removed with it. A key that
  fails to decrypt (for example after a master key change) stays stored, and a
  bundle whose key was superseded by a newer issuance is refused.
- Installation-wide audit events (users created, changed or deleted,
  organizations and API keys deleted, server certificates generated or
  renewed) are filed under the platform tenant `general.tenant_id`, and only
  administrators read them: the event list, event stream, error groups and the
  base station and endpoint activity timelines all apply the caller's
  readable event categories.
- Integrations are managed by administrators only, and their settings are
  write-only: `GetIntegration`, `ListIntegrations` and the create and update
  responses name each configured setting with the value `configured` instead
  of returning credentials such as HTTP headers or broker passwords.
- An administrator's password reset (`UpdateUserPassword`) signs the user out
  everywhere: every refresh token issued before the reset is revoked, as a
  user's own password change already did. A reset of a compromised account
  left the attacker's session renewable. The reset reports an error when the
  sessions cannot be revoked.
- A base station's answer to a `dlDataRev` ends only a downlink that station
  holds. A late confirmation or refusal from a station that no longer held the
  downlink revoked it while another station held it, so that station's
  transmission was reported to nobody. The revoke of a downlink still in the
  queue is unchanged.
- An Application Center's `queId` names one downlink of its organization until
  the downlink's result. It was unique within the tenant for good, so a
  `dlDataQue` failed with `EEXIST` when another organization of the tenant had
  used the id, which disclosed that organization's id, and an id could never be
  used again after its downlink ended. Migration 000172 scopes the id to the
  organization's downlinks in flight; two in-flight downlinks of one
  organization still refuse to share an id. The down migration refuses while an
  id repeats within a tenant.
- A base station's client certificate is bound to the station it claims in
  every edition, not only with organization enforcement. Before, a community
  edition base station with any certificate the CA signed could connect as
  another registered station and receive its endpoints' network session keys
  and queued downlinks. A certificate naming another station, or one that is
  not the certificate pinned for the station, is now refused as if the station
  were not registered. A station registered without a certificate is pinned to
  the first certificate that names it.
- The unauthenticated CE onboarding call completes onboarding exactly once:
  two concurrent calls on an installation that had not finished onboarding
  both succeeded, and the later one overwrote the company name.
- A SCACI `regCmp` reaches only the registering Application Center's own
  tenant. A `reg` is recorded before it is validated, so a refused `reg`
  naming another tenant's EUI followed by `regCmp` found that tenant's
  endpoint and, when it was pre-attached, attached it and propagated its keys.
- A SCACI `deregCmp` completes only a `dereg` the service center answered with
  `deregRsp`, and detach propagation sends `detPrp` only for an endpoint of
  the deregistering tenant. A `dereg` is recorded before it is validated, so a
  refused `dereg` naming another tenant's EUI followed by `deregCmp` sent that
  endpoint `detPrp` on every base station; any recorded operation carrying an
  `epEui` did the same. A `deregCmp` for a `dereg` answered with an error, one
  not answered yet or already completed, another operation or an unknown
  `opId` is now answered with `scaci.error.unexpected_deregister_complete`
  (EPROTO) and neither revokes downlinks nor sends `detPrp`. The completion
  reads the `dereg` from the operation log, so a `deregCmp` after a resume
  completes it too.
- A SCACI session is resumed only by the Application Center it belongs to:
  the same organization and `acEui`. The resume compared only the tenant, so
  an Application Center of another organization of the same tenant that
  presented a disconnected session's `snAcUuid` took that session over under
  its organization, closed its connection, received the downlink results
  held for it and queued downlinks in its name. Such a connect now starts a
  new session of its own organization.
- A new SCACI session supersedes only the earlier sessions of its own
  tenant, organization and `acEui`. An Application Center of another
  organization of the same tenant that claimed an `acEui` in use closed the
  owner's live connection, discarded what was held for its disconnected
  session and ended that session's resumability. Each organization now keeps
  its own session of the `acEui`: migration 000171 allows one live session
  per Application Center of an organization instead of one per tenant and
  `acEui`, which failed such a connect with an internal error while the owner
  was live and locked the owner out while it was offline; and a resume looks
  up only the connecting Application Center's session, so another
  organization's session of the owner's `snAcUuid` no longer defeats the
  owner's resume. The down migration refuses while an `acEui` is live in more
  than one organization of a tenant.
- A SCACI `dlDataRes` reaches only the Application Center that queued the
  downlink. It was sent to the Application Center whose `dlDataQue` last used
  the same queue id anywhere in the tenant, so another Application Center,
  also of another organization, that reused the queue id received it, and
  every session of the queuer's `acEui` in any organization of the tenant,
  connected or held for resume, received it too. A `dlDataQue` is now
  recorded with the downlink's service center queue id (`scQueId`), the
  result is routed by that id inside the downlink's organization, and only
  that organization's Application Center of the `acEui` is sent it.
  **Upgrade:** a downlink an Application Center queued before the upgrade and
  still in flight reports its result on MQTT and in the events only, since
  its recorded `dlDataQue` carries no service center queue id.
- A SCACI or BSSCI peer certificate is resolved to its tenant from the
  organization directory on every connection. It was read from the request
  cache, so a peer of an organization moved to another tenant or deleted kept
  its old tenant for up to the cache TTL. Per-request organization lookups
  keep the cache.
- The failure event of an attach or detach propagation that finds no
  connected base station names the endpoint, and is now recorded for the
  tenant that owns it. It was recorded for the server's default tenant, which
  was shown other tenants' endpoint EUIs; an endpoint no tenant owns is now
  only logged.
- An internal gRPC hop other hosts can reach requires the internal peer
  secret (`internal_auth.peer_secret`, `KILOCENTER_INTERNAL_AUTH_PEER_SECRET`,
  at least 32 characters). KC-Core and KC-Identity trusted the gateway's
  identity headers from any peer when the secret was empty, the default, so
  anything that reached port 50051 or 50052 could act as any tenant and user.
  KC-Core now refuses to start with `grpc.internal_trust_enabled` on a
  non-loopback `grpc.host` (an empty host binds every interface) or a
  non-loopback `identity.address` without the secret; KC-Identity with a
  non-loopback `grpc.host`; KC-Gateway with a non-loopback upstream.
  **Upgrade:** Docker Compose now requires `KILOCENTER_INTERNAL_PEER_SECRET`
  in `.env` and the Helm chart requires `secrets.internalPeerSecret`
  (`openssl rand -hex 32`, the same value for all three services); a
  single-host install on `localhost` needs nothing.

### Fixed
- Creating or updating an endpoint with an all-zero network session key is
  refused with `INVALID_ARGUMENT` (`KC-GRPC-ERR-282`). Such a key can never be
  sent to a base station, and the endpoint used to be stored and then fail
  to attach with an internal error.
- Deleting an endpoint no longer logs an error when the endpoint is removed
  before its detach is recorded for the base stations.
- Certificates from `certgen` (the CA, the service center and every base
  station) no longer claim a country, province and locality of San Francisco,
  US; they name the product and the holder only.
- A request the service center refuses (invalid input, not found, not
  permitted) and a remote peer that fails the TLS handshake or drops the link
  are logged as warnings with the peer's address, not as errors, so errors
  point at faults of the service center itself.
- Downloading a base station's private key no longer hangs until the request
  times out: the key's row lock blocked the audit record of its own download.
- An uplink whose receptions by several base stations arrive at the same time
  records the endpoint's acknowledgement of its downlink once; two receptions
  could both record it, and the event appeared twice.
- A stored base station private key that cannot be decrypted, for example
  under a wrong master key, is answered with an internal error instead of
  "certificate not found", which read as a key already downloaded. The key
  stays stored and can be downloaded once the master key is correct.
- A base station location outside the globe (latitude beyond ±90, longitude
  beyond ±180) is refused with `INVALID_ARGUMENT` and a message naming the
  valid range, instead of an internal error.
- A retried SCACI uplink delivery no longer reaches an Application Center a
  second time. When one Application Center session did not get an uplink, the
  delivery worker retried it for all of them, and every attempt sent it again
  under a new operation id with `duplicate: false`, so a session that had it
  already processed the telegram twice. An Application Center session now
  holds one `ulData` operation per stored uplink: a retry reaches only the
  sessions that lack it, a held session counts it once toward
  `protocol.scaci_resume_max_pending_operations`, and a session whose
  connection fails while the uplink is written keeps it for its resume, which
  reissues it under its original operation id (SCACI §1, §3.2, §3.8.1).
  Migration 000186 adds the unique index that keeps one such record per
  session and uplink; the upgrade preflight counts the rows its build scans.
- One uplink received by several base stations reaches the Application
  Centers and MQTT as one `ulData` listing every receiving station. The
  delivery was due the moment the first reception was stored, so a second
  station's report that arrived after the next delivery poll was stored with
  the message but never delivered. A new uplink is now delivered once the new
  `protocol.delivery.reception_window` has passed (default 500ms; Helm
  `kcCore.config.protocol.delivery.receptionWindow`), which must be shorter
  than `protocol.duplicate_window`.
- The live uplink and event streams no longer drop a row stored after a newer
  one has streamed but carrying an older time (a skewed station clock, a
  relayed or replayed uplink, a refused SCACI connect moved forward), nor rows
  sharing a timestamp. They read in the order the database stored the rows,
  reaching back by the new `grpc.stream_overlap` (default 10s). Migration
  000185 indexes the ulData rows of `messages` by storage time and adds
  `system_events.stored_at`; the upgrade preflight counts the rows the index
  builds cover.
- An uplink a base station received as a secondary receiver opens from that
  station's message list with its own RSSI and SNR instead of answering
  `NOT_FOUND`.
- Event listings include the events at both bounds of their time window.
- A new base station is no longer handed a Service Center URL it cannot
  reach. When `protocol.bsci_external_url` (or, without it,
  `protocol.bsci_host`) names a wildcard or loopback address such as
  `tls://localhost:5000`, the certificate bundle and the station carry no
  URL, and the Add Base Station result and certificate views warn the
  operator to set `protocol.bsci_external_url`
  (`KILOCENTER_PROTOCOL_BSCI_EXTERNAL_URL`).
- Renewing the server certificate keeps the names base stations connect by:
  the host of `protocol.bsci_external_url`, else the current certificate's
  subject, plus the current certificate's names and
  `certificates.server_names`. Before, renewal without an external URL named
  the certificate `localhost`, and certgen added every address of the host's
  interfaces, publishing the host's internal network layout.
- The BSSCI and SCACI listeners serve a renewed server certificate from the
  next connection on; a renewal no longer needs a KC-Core restart.
- A regenerated base station certificate is offered for download in the edit
  dialog, as the Add Base Station wizard offers a new one, and its event shows
  on the station's activity page.
- Activity date filters keep the date the admin typed, accept digits without
  separators and cover whole local days.
- A base station whose certificate was issued outside the service center is
  warned before regeneration that it is refused until it has the new files.
- A dropped gRPC-web stream behind the development proxy now ends in the
  browser too, so it reconnects instead of holding a connection.
- A downlink whose revoke the holding base station answers with an error,
  such as the AVA station's code 95 "no matching DL data found" for a
  downlink it no longer holds, now ends `revoked` instead of staying `queued`
  forever: the station will never transmit it. The revocation is recorded as
  a confirmed one is. A downlink that already ended keeps its outcome, so one
  the expiry sweep expired stays `expired` and is not reported again.
- A base station that resumes its session keeps the downlinks it had queued:
  it is sent attPrp only for the endpoints attached after its connection was
  lost, since an attPrp for an endpoint it holds makes it discard that
  endpoint's queued downlinks. A KC-Core restart or deploy no longer strands
  them until they expire. Every attach decision, including an over-the-air
  attach of an endpoint that was already attached, now records its time, so
  migration 000170 renames `endpoints.status_changed_at` to
  `attachment_changed_at` and corrects its comment; the values carry over. A
  new session, or a resumed one whose disconnect time is unknown, is still
  sent every attached endpoint.
- A downlink a base station held for an endpoint is no longer stranded as
  `queued` when the station is sent an attPrp for that endpoint (an API
  attach of an attached endpoint, an over-the-air attach heard by another
  station, or reconnection): once the station answers the attPrp, the
  downlinks it had queued for the endpoint return to pending, with no result
  reported, and go out again at the endpoint's next downlink window. A
  downlink sent to the station after the attPrp stays queued there.
- A downlink queued while the base station serving its endpoint was offline
  is sent to that station as soon as it completes its connect, new session or
  resumed, after the attachments it is sent on connect, instead of waiting
  for the endpoint's next uplink to open a downlink window. Only the downlinks
  the station serves when it connects are sent this way; a window an uplink
  opens afterwards is filled by the in-window dispatch, which still carries
  at most one downlink.
- Uplinks are stored over the packet counter's full 32-bit range (BSSCI
  §3.10.1); a counter from 2^31 on failed to persist. Migration 000167 widens
  `messages.packet_cnt` and `messages_archive.packet_cnt` to BIGINT, which
  rewrites both tables during the upgrade.
- An Application Center that reconnects after losing its connection can open
  a new SCACI session again (the earlier session no longer blocks it).
- Uplinks with empty `userData` are delivered over SCACI.
- A response to no outstanding service center operation is answered with a
  SCACI error instead of a completion.
- An uplink reissued after a SCACI resume carries its full reception details
  (`rxDuration`, `eqSnr`, `dlRxSnr`, `dlRxRssi`, `profile`, `mode`,
  `subpackets`) and its `duplicate` flag.
- A slow resume write of a SCACI connection that a newer connection of the
  same session already replaced can no longer land after that connection's
  loss and leave the stored session active without a connection: a session's
  resume and connection-loss writes follow the order of the moves.
- A resumed SCACI session whose connection drops right after `conCmp` is
  recorded as disconnected instead of active, and a resume never revives a
  session that is no longer resumable. A second `conCmp` is answered with an
  error.
- Organization look-ups report "not found" instead of an internal error, and a
  failed repository transaction always releases its connection.
- Down migrations 000145, 000154 and 000155 restore the exact previous schema.
- Downlinks without a payload (pure acknowledgements) can be queued.
- An uplink with `dlAck` relayed by federation acknowledges the endpoint's
  downlink of the previous window, as a base station's own ulData does.
- An Application Center may queue a downlink under the queue id 0, which
  SCACI allows; it was refused with `scaci.error.que_id_zero`, and its result
  could not be sent. Migration 000169 lets `downlink_queue.ac_que_id` hold 0.
- dlDataQue `userData` in the SCACI Numeric[m][n] shape is accepted.
- A counter-dependent downlink keeps its user data when it is listed,
  dispatched to a base station or reported; its stored entries were dropped
  on every read.
- Revoking a downlink the base station holds no longer fails with a database
  constraint error.
- Certificate bundle downloads are limited to the caller's tenant and no longer
  contain the CA private key; expired bundles are removed.
- The Certificates page reports the CA's subject, issuer and days until expiry.
- A normal shutdown no longer logs `gRPC server failed`.
- A fractional uplink `eqSnr` (for example 19.8 dB) is stored and forwarded to
  Application Centers instead of being dropped.
- Subpacket frequencies that base stations report with fractions of a hertz are
  stored for uplinks, attaches and detaches instead of being dropped.
- An MQTT downlink command the service center refuses (invalid, oversized, or for
  an unknown endpoint) is reported on `event/downlink_rejected` instead of being
  dropped silently.
- A SCACI `reg` of an endpoint the service center does not know creates it, as
  class A when `bidi` is set and class Z otherwise, instead of failing with
  `failed_create_endpoint`.
- A SCACI `regCmp` completes only a `reg` the service center answered with
  `regRsp`. One for a `reg` answered with an error, an already completed
  `reg`, another operation or an unknown `opId` is answered with
  `scaci.error.unexpected_register_complete` (EPROTO) and attaches nothing.
- A SCACI `regCmp` with `preAttach` attaches and propagates the registered
  endpoint. The operation log read real EUIs (above 2^53) back rounded, so the
  endpoint was not found.
- The first over-the-air attach of an endpoint is accepted with any attach
  counter, 0 included; only later attaches must advance the counter.
- An attach or detach that does not report `eqSnr` leaves the endpoint's
  recorded eqSnr unchanged instead of recording the reported `snr` as eqSnr.
- The same detach reported by two base stations is announced (`epStat`, MQTT)
  and propagated once.
- An attach or detach completed after a BSSCI session resume forwards the
  endpoint's nonce and signature in `epStat`.
- `ListDownlinkQueue` reports the base station that holds a queued downlink in
  `bs_eui` instead of `0000000000000000`, and both it and `GetDownlinkResults`
  leave `bs_eui` empty when no station holds the downlink.
- `start-dev.sh` stops the services a previous run started before starting
  new ones; its patterns matched none of them and it never stopped
  KC-Identity. The new `dev-services.sh` starts each service with a pid file
  holding the service's own PID and `dev-services.sh stop` stops exactly
  those processes and what they started (the vite server under `bun run
  dev`), KC-Web and KC-Gateway before KC-Identity and KC-Core, waiting for
  each to exit. `stop-all-services.sh` uses it and no longer kills every
  process matching `bun.*dev`.

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

[2.0.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.3.0...v2.0.0
[1.3.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/Kiloiot/kilo-service-center/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Kiloiot/kilo-service-center/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Kiloiot/kilo-service-center/releases/tag/v1.0.0
