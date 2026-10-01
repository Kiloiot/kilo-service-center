# MQTT First Steps

## Goal

Enable MQTT integration and validate event consumption and command publishing.

## Prerequisites

MQTT is **disabled by default** in KiloCenter. Before using MQTT:

1. Ensure the Mosquitto broker is running (included in the Docker Compose stack).
2. Enable MQTT in your KiloCenter configuration:

```yaml
mqtt:
  enabled: true
  host: "localhost"
  port: 1883
  username: "admin"
  password: "KiloCenter"
  topic_prefix: "mioty"
  enable_command_subscriptions: false
```

> **Note:** Set `enable_command_subscriptions: true` if you want KiloCenter to accept downlink commands via MQTT. Restart KC-Core after any configuration change.

3. Restart KC-Core for the change to take effect.

## Local Broker Defaults

| Setting | Value |
|---------|-------|
| Broker (TCP) | `localhost:1883` |
| Broker (WebSocket) | `localhost:9001` |
| Username | `admin` |
| Password | `KiloCenter` |
| Topic prefix | `mioty` (configurable in `config.yaml`) |

## MQTT Wildcards

MQTT supports two wildcard characters in subscription topics:

- `+` matches a single topic segment (e.g., `mioty/+/device/+/event/up` matches any org and device).
- `#` matches all remaining segments (e.g., `mioty/<ORG_UUID>/device/#` matches all device topics for one org).

Topics are case-sensitive.

## Topic Contract

All MQTT topics follow this format:

```text
{prefix}/{org_uuid}/device/{ep_eui_hex}/{channel}/{type}
```

- `{prefix}` defaults to `mioty`.
- `{org_uuid}` is your organization UUID.
- `{ep_eui_hex}` is the 16-character lowercase hex EUI of the endpoint (e.g., `70b3d59cd00009e6`).

### Event Topics (Published by KiloCenter)

```text
{prefix}/{org_uuid}/device/{ep_eui_hex}/event/{event_type}
```

Supported `event_type` values:

| Event type | Description |
|------------|-------------|
| `up` | Uplink data received from a device |
| `attach` | Endpoint attached to a base station |
| `detach` | Endpoint detached from a base station |
| `downlink_queued` | A `command/down` downlink was accepted and queued |
| `downlink_rejected` | A `command/down` downlink was refused, with the reason |
| `downlink_result` | Result of a queued downlink, and the endpoint's acknowledgement of it |

### Command Topic (Consumed by KiloCenter)

```text
{prefix}/{org_uuid}/device/{ep_eui_hex}/command/down
```

## Subscribe to Events

Subscribe to all events for one organization:

```bash
mosquitto_sub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/+/event/+' -v
```

Subscribe to uplinks only:

```bash
mosquitto_sub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/+/event/up' -v
```

Subscribe to a single device:

```bash
mosquitto_sub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/+' -v
```

## Publish a Downlink Command

```bash
mosquitto_pub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/<EP_EUI_HEX>/command/down' \
  -m '{"data":"AQIDBA==","confirmed":false}'
```

Replace `<ORG_UUID>` with your organization UUID and `<EP_EUI_HEX>` with your endpoint's EUI in hex format.

## Observe Downlink Outcomes

Every downlink command you publish is answered on `event/downlink_queued` or `event/downlink_rejected`, and a queued downlink later reports on `event/downlink_result`:

```bash
mosquitto_sub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/+' -v
```

## Message Payloads

### `event/up`

```json
{
  "epEui": "70b3d59cd00009e6",
  "bsEui": "0011223344556677",
  "rssi": -95,
  "snr": 7.5,
  "eqSnr": 6.9,
  "rxTime": 1737025800000000000,
  "rxDuration": 250000000,
  "cnt": 1234,
  "data": "ACk=",
  "format": 0,
  "profile": "eu868",
  "mode": "ulp",
  "subpackets": { "snr": [7.1, 7.9], "rssi": [-95.2, -94.8], "frequency": [868180000, 868220000] },
  "dlOpen": true,
  "responseExp": false,
  "dlAck": false,
  "baseStations": [
    { "bsEui": "0011223344556677", "rxTime": 1737025800000000000, "snr": 7.5, "rssi": -95, "eqSnr": 6.9 },
    { "bsEui": "8899aabbccddeeff", "rxTime": 1737025800000000100, "snr": 2.1, "rssi": -118.4 }
  ],
  "decodedPayload": { "temperature": 41 },
  "decodeStatus": "success",
  "blueprintTypeEui": "70b3d56770110000"
}
```

- `epEui` is the endpoint; it is also the device segment of the topic.
- `bsEui`, `rssi`, `snr`, `eqSnr`, `rxTime`, `rxDuration`, `profile`, `mode` and `subpackets` describe the first reception; `baseStations` lists every base station that received the uplink, each with its own metrics, and a base station that reported downlink reception quality adds `dlRxSnr` and `dlRxRssi`.
- `data` is base64-encoded payload bytes; `format` is the uplink's payload format number.
- `rxTime` and `rxDuration` are nanoseconds. `rxTime` is larger than 2^53, so a JavaScript client that needs every nanosecond must read it with a big-integer JSON parser.
- `subpackets` holds the per-subpacket `snr` (dB), `rssi` (dBm), `frequency` (Hz) and, when the base station reports it, `phase` (degrees).
- `dlOpen` is `true` when the endpoint opened a downlink window after this uplink; a queued downlink can be sent in it.
- `responseExp` is `true` when the endpoint expects a response in that window.
- `dlAck` is `true` when the endpoint acknowledges the downlink it received in its previous window.
- `duplicate` is `true` when the endpoint reused a packet counter, with `packetCntReused` when that happened outside the duplicate window.
- `decodeStatus` is `success`, `failed`, `skipped` (no blueprint) or `pending`. On success `decodedPayload` holds the decoded values and `blueprintTypeEui` names the blueprint's device type; on failure `decodeErrorCode` names the reason.

### `event/attach`

```json
{
  "epEui": "70b3d59cd00009e6",
  "bsEui": "0011223344556677",
  "event": "attach"
}
```

### `event/detach`

```json
{
  "epEui": "70b3d59cd00009e6",
  "bsEui": "0011223344556677",
  "event": "detach"
}
```

Each attachment and detachment is published once. `bsEui` names the base station that heard an over-the-air attach or detach; an attach or detach made in KC-Web, through the API or by an Application Center has no `bsEui`.

### `event/downlink_queued`

```json
{
  "epEui": "70b3d59cd00009e6",
  "queId": 4503599627370497,
  "ref": "order-17"
}
```

- `queId` identifies the downlink in the later `event/downlink_result`. It is always at most 2^53-1, so every JSON client reads it exactly.
- `ref` echoes the `ref` of your command and is omitted when the command had none. KiloCenter stores it with the downlink, so every later `event/downlink_result` of the downlink carries it too.

### `event/downlink_rejected`

```json
{
  "epEui": "70b3d59cd00009e6",
  "code": "mqtt.command.payload_too_large",
  "message": "decoded data exceeds the 200-byte radio payload maximum",
  "ref": "order-18"
}
```

`code` is stable and meant for your client to match on; `message` explains it. Refusals detected in the command itself:

| `code` | Reason |
|--------|--------|
| `mqtt.command.empty_payload` | The MQTT message is empty |
| `mqtt.command.message_too_large` | The MQTT message exceeds 1 MB |
| `mqtt.command.invalid_json` | The message is not a JSON object |
| `mqtt.command.invalid_field` | A field has the wrong type or is out of range, such as `format` above 255 |
| `mqtt.command.missing_data` | Neither `data` nor `entries` is present |
| `mqtt.command.data_with_entries` | Both `data` and `entries` are present |
| `mqtt.command.empty_entries` | `entries` is an empty array |
| `mqtt.command.missing_packet_cnt` | An entry has no `packetCnt` |
| `mqtt.command.duplicate_packet_cnt` | Two entries name the same `packetCnt` |
| `mqtt.command.invalid_base64` | A `data` value is not valid base64 |
| `mqtt.command.payload_too_large` | A decoded payload exceeds 200 bytes |
| `mqtt.command.org_unresolved` | The organization in the topic does not exist; a command with a `ref` whose organization could not be looked up at all is not answered (see **A `ref` is accepted once** below) |
| `mqtt.command.enqueue_failed` | The service center could not queue the downlink |
| `mqtt.command.ref_too_long` | `ref` exceeds 128 bytes |
| `mqtt.command.expired` | `expiresAt` passed before the downlink could be queued |

Refusals by the service center core carry its catalog token, for example `scaci.error.endpoint_not_found` for an endpoint that is not registered to you.

A message on a malformed topic (an invalid organization UUID or endpoint EUI) cannot be answered, because it has no event topic; KiloCenter only logs it.

### `event/downlink_result`

```json
{
  "epEui": "70b3d59cd00009e6",
  "queId": 4503599627370497,
  "result": "sent",
  "bsEui": "70b3d59cd00009e6",
  "txTime": 1737025801000000000,
  "packetCnt": 1235,
  "ref": "order-17"
}
```

Possible `result` values:

| `result` | When it is published |
|----------|----------------------|
| `sent` | A base station transmitted the downlink |
| `expired` | The downlink reached its deadline before it was sent: still in the service center queue, or at a base station that then dropped it, said it does not hold it or started a new session; or the base station holding it was deleted (see **Deadline** below) |
| `invalid` | A base station refused the downlink |
| `acknowledged` | The endpoint confirmed it received the transmitted downlink |

- Only a `sent` result carries `bsEui` (the base station that transmitted the downlink) and `txTime` (Unix time of transmission in nanoseconds).
- `packetCnt` is the endpoint packet counter of the downlink window: the window the downlink was sent in for `sent`, and the window the endpoint acknowledged for `acknowledged`.
- `ref` is the `ref` of the `command/down` that queued the downlink, on every result, and is omitted when the command had none. A downlink queued by an Application Center or through the API has no `ref`.

`sent`, `expired` and `invalid` are final: each downlink reports exactly one of them, which, like every downlink outcome, can arrive twice (see [QoS Behavior](#qos-behavior)). `acknowledged` follows a `sent` result when the endpoint's next uplink sets the downlink acknowledgement flag (`dlAck`, which also appears on `event/up`) for the window the downlink was sent in. It is published once per transmitted downlink: a repeated reception of that uplink, an uplink whose acknowledgement matches no transmitted downlink, and an uplink without the flag publish nothing. When the endpoint's packet counter restarted and reused a window, the downlink transmitted last in that window is the one acknowledged. A downlink the endpoint never acknowledges publishes no `acknowledged`, so absence after `sent` means the endpoint has not confirmed it.

KiloCenter stores the acknowledgement together with its record and publishes it from its delivery outbox, the way it publishes uplinks: an acknowledgement recorded while your broker is unreachable, or carried by an uplink relayed from a federated Community Edition, is published once the broker accepts it. Like an uplink, it can arrive a second time when KiloCenter restarts between publishing it and recording that it did, so treat a repeated `acknowledged` for the same `queId` as the same confirmation.

```json
{
  "epEui": "70b3d59cd00009e6",
  "queId": 4503599627370497,
  "result": "acknowledged",
  "packetCnt": 1235,
  "ref": "order-17"
}
```

### `command/down` (You Publish This)

The minimal command:

```json
{
  "data": "AQIDBA==",
  "confirmed": false
}
```

Every other field is optional:

```json
{
  "data": "AQIDBA==",
  "confirmed": true,
  "prio": 2.5,
  "format": 7,
  "responsePrio": false,
  "dlWindReq": false,
  "expOnly": false,
  "dlRxStatQry": false,
  "ref": "order-17",
  "expiresAt": "2026-10-01T12:00:00Z"
}
```

A downlink that depends on the endpoint packet counter, for example because of application encryption, gives one payload per counter in `entries` instead of `data`; it is sent only in the downlink window of a listed counter:

```json
{
  "entries": [
    { "packetCnt": 1235, "data": "AQ==" },
    { "packetCnt": 1236, "data": "Ag==" }
  ],
  "ref": "order-18"
}
```

An empty `data` queues a pure acknowledgement downlink:

```json
{ "data": "" }
```

| Field | Type | Meaning |
|-------|------|---------|
| `data` | base64 string | The downlink payload; empty for a pure acknowledgement |
| `entries` | array of `{packetCnt, data}` | Counter-dependent payloads; mutually exclusive with `data` |
| `confirmed` | boolean | Request a response from the endpoint (`responseExp`) |
| `prio` | number | Priority, higher values first (single precision, default 0) |
| `format` | integer 0-255 | User data format identifier |
| `responsePrio` | boolean | Request a priority response from the endpoint |
| `dlWindReq` | boolean | Request a further downlink window from the endpoint |
| `expOnly` | boolean | Send only when the endpoint expects a response |
| `dlRxStatQry` | boolean | Ask the endpoint for its downlink reception status |
| `ref` | string | Your correlation id of at most 128 bytes, echoed in `downlink_queued`, `downlink_rejected` and every `downlink_result` of the downlink |
| `expiresAt` | RFC 3339 string | The latest time the downlink may be transmitted, for example `2026-10-01T12:00:00Z` or with fractional seconds and an offset |

Validation rules:
- Exactly one of `data` and `entries` is present.
- Every `data` value is valid base64 and decodes to at most 200 bytes, the radio downlink payload limit.
- Every entry has a `packetCnt` (0 to 4294967295), and no counter repeats.
- `ref`, when present, is at most 128 bytes.
- `expiresAt`, when present, is an RFC 3339 time; anything else is refused with `mqtt.command.invalid_field`.
- Raw MQTT payload maximum: 1 MB.
- A refused command is reported on `event/downlink_rejected`; nothing is queued.

**Deadline.** A downlink waits for its endpoint's downlink window until `protocol.downlink_expiry.lifetime` has passed, or until `expiresAt` when that comes first. A downlink still in the service center queue at its deadline ends `expired` at once. A downlink a base station already holds may have been transmitted in the meantime, so KiloCenter asks the station to drop it and reports only what it has reason to believe: `sent` when the station reports the transmission, and `expired` in one of these cases:

- the downlink was still in the service center queue when its deadline passed, so no station ever held it;
- the holding station confirmed that it dropped the downlink, or answered that it does not hold it (`protocol.downlink_expiry.revoke_not_held_codes`);
- the holding station opened a new session that is not a resumed one, which discards everything the previous session held, a pending result included (BSSCI §1);
Until one of these happens nothing is published, also while the station is offline or keeps refusing the revoke with another code; a connected station is asked again once per `protocol.downlink_expiry.sweep_interval`, and an offline one when it reconnects. Time passing alone never ends a downlink a connected station holds. When the station that held a downlink reports `sent` after it was reported `expired`, it contradicts the report: the result stays `expired`, nothing is published again, and the service center logs a warning and records one `dl_data_sent_after_expiry` event for the endpoint, however often the station repeats it.

**Deleting a base station ends every downlink it held as `expired`**, before or after its deadline: the ones it holds queued, the ones reserved for it and the ones it is asked to drop. Its session is closed first, and a deleted station never connects again, so it can no longer report what became of them, and none is returned to the queue, where the endpoint could receive it a second time from another station. A station that is still powered may nevertheless transmit what it held after it was deleted, so the endpoint may receive a downlink reported `expired`. To avoid that, power the station off, or wait until no downlink is queued at it, before you delete it.

A revoke requested by an operator or an Application Center for a downlink whose deadline has passed while a station holds it ends `expired`, not revoked: the station's answer settles the expiry already under way.

A command whose `expiresAt` has already passed when KiloCenter receives it queues nothing and is refused with `mqtt.command.expired`.

**A `ref` is accepted once.** KiloCenter keeps every `ref` an organization used for an endpoint, so you can safely publish a command again when you are not sure it arrived, for example after your client restarted. A command whose `ref` already queued a downlink for the same endpoint queues nothing and publishes nothing: neither `downlink_queued` nor `downlink_rejected`. The `ref` is compared as soon as the organization is resolved, before the payload and the endpoint are checked, so this holds while the first downlink is waiting and after it ended, also when the repeat's `expiresAt` has passed by then, or the endpoint was deleted or lost its downlink capability since: a repeat is never refused for something the first command passed. A field of the wrong type in a repeat does not get it refused either, as long as the `ref` itself is readable. If the service center cannot look the `ref` up, or cannot look up the organization of a command with a `ref`, it does not answer the command at all rather than risk refusing one it accepted; publish it again later. The first command's `downlink_queued` and its `downlink_result` are the outcome. Use a new `ref` for every new downlink; commands without a `ref` are never compared.

## Copy-Paste Cookbook

Set shell variables once to simplify repeated commands:

```bash
export MQTT_HOST=localhost
export MQTT_PORT=1883
export MQTT_USER=admin
export MQTT_PASS=KiloCenter
export MQTT_PREFIX=mioty
export ORG_UUID=<your-org-uuid>
export EP_EUI_HEX=<your-endpoint-eui>
```

Then use them in any command:

```bash
# All events for one org
mosquitto_sub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "$MQTT_PREFIX/$ORG_UUID/device/+/event/+" -v

# Single device events
mosquitto_sub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "$MQTT_PREFIX/$ORG_UUID/device/$EP_EUI_HEX/event/+" -v

# Send a downlink
mosquitto_pub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "$MQTT_PREFIX/$ORG_UUID/device/$EP_EUI_HEX/command/down" \
  -m '{"data":"AQIDBA==","confirmed":false}'
```

## End-to-End Flows

### Watch Device Traffic

1. Start a subscriber for your org:
   ```bash
   mosquitto_sub -h localhost -p 1883 \
     -u admin -P KiloCenter \
     -t "mioty/<ORG_UUID>/device/+/event/+" -v
   ```
2. When a device sends an uplink, you receive a message on `.../event/up` with radio metrics and payload.

### Send a Downlink via MQTT

1. Publish a command:
   ```bash
   mosquitto_pub -h localhost -p 1883 \
     -u admin -P KiloCenter \
     -t "mioty/<ORG_UUID>/device/<EP_EUI_HEX>/command/down" \
     -m '{"data":"AQIDBA==","confirmed":false}'
   ```
2. KiloCenter resolves the org and validates the command. You receive `.../event/downlink_queued` with the `queId`, or `.../event/downlink_rejected` with the reason.
3. When a base station sends the downlink, or it expires, you receive `.../event/downlink_result` with the same `queId` and your `ref`.
4. When the endpoint confirms it received a sent downlink, you receive a second `.../event/downlink_result` with `"result": "acknowledged"`.

### Observe Attach/Detach Lifecycle

1. Subscribe to a single device:
   ```bash
   mosquitto_sub -h localhost -p 1883 \
     -u admin -P KiloCenter \
     -t "mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/+" -v
   ```
2. When the endpoint attaches or detaches, you receive `.../event/attach` or `.../event/detach`.

## QoS Behavior

| Topic | QoS |
|-------|-----|
| `event/up` | 1 (at least once) |
| `event/downlink_queued` | 1 (at least once) |
| `event/downlink_rejected` | 1 (at least once) |
| `event/downlink_result` | 1 (at least once) |
| `event/attach` | 0 (at most once) |
| `event/detach` | 0 (at most once) |
| `command/down` subscription | 1 (at least once) |

The downlink outcomes are published at least once because a platform settles its commands on them: none is lost while the broker session holds, but any of them can arrive twice. Treat a repeated `downlink_queued` or `downlink_result` for the same `queId` and result, or a repeated `downlink_rejected` for the same `ref`, as the same outcome. Uplinks can arrive twice for the same reason, so make your client idempotent for every QoS 1 topic.

## Tenant Isolation

KiloCenter enforces data isolation through two layers:

**Application-level isolation** -- KiloCenter resolves endpoint ownership to the correct tenant and organization before publishing. MQTT topics are constructed from the owner's org UUID. If the owner org cannot be resolved, the publish is skipped entirely rather than risking data leakage.

**Broker-level isolation** -- You should enforce per-organization topic ACL rules at the MQTT broker. Without ACL rules, a wildcard subscriber could see events from multiple organizations.

### Recommended ACL Pattern

Create one broker user per organization or integration and restrict topic access.

Example (Mosquitto syntax):

```conf
user org-a-integration
topic read  mioty/<ORG_A_UUID>/device/+/event/#
topic write mioty/<ORG_A_UUID>/device/+/command/down
```

Do not grant broad wildcard access like `mioty/+/device/+/event/#` to tenant-specific clients.

## Troubleshooting

### No Events Received

1. Verify your subscribe topic matches your org UUID exactly.
2. Confirm ACL allows read access to your org's event topics.
3. Check that the device actually produced traffic or a state transition.
4. Verify MQTT broker connectivity and credentials.

### Downlink Command Has No Effect

1. Confirm `mqtt.enable_command_subscriptions` is set to `true` in your configuration.
2. Subscribe to `mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/downlink_rejected`; a refused command is reported there with its reason.
3. Verify the topic is exactly `mioty/<ORG_UUID>/device/<EP_EUI_HEX>/command/down`.
4. Check that ACL allows write access to your org's command topic.

### Quick Broker Sanity Check

Subscribe to all topics to verify the broker is receiving messages:

```bash
mosquitto_sub -h localhost -p 1883 -u admin -P KiloCenter -t 'mioty/#' -v
```
