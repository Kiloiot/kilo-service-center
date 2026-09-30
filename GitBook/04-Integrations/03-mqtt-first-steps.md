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
| `downlink_result` | Result of a queued downlink |

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
  "bsEui": "0011223344556677",
  "rssi": -95,
  "snr": 7.5,
  "rxTime": 1737025800000000000,
  "cnt": 1234,
  "data": "SGVsbG8=",
  "dlOpen": true,
  "responseExp": false,
  "dlAck": false
}
```

- `data` is base64-encoded payload bytes.
- `rxTime` is a Unix timestamp in nanoseconds. It is larger than 2^53, so a JavaScript client that needs every nanosecond must read it with a big-integer JSON parser.
- `dlOpen` is `true` when the endpoint opened a downlink window after this uplink; a queued downlink can be sent in it.
- `responseExp` is `true` when the endpoint expects a response in that window.
- `dlAck` is `true` when the endpoint acknowledges the downlink it received in its previous window.
- A payload that also has a matching blueprint carries a `decodedPayload` object.

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
- `ref` echoes the `ref` of your command and is omitted when the command had none.

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
| `mqtt.command.org_unresolved` | The organization in the topic cannot be resolved |
| `mqtt.command.enqueue_failed` | The service center could not queue the downlink |

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
  "packetCnt": 1235
}
```

Possible `result` values: `sent`, `expired`, `invalid`. Only a `sent` result carries `bsEui` (the base station that transmitted the downlink), `txTime` (Unix time of transmission in nanoseconds) and `packetCnt` (the endpoint packet counter of the window it was sent in).

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
  "ref": "order-17"
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
| `ref` | string | Your correlation id, echoed in `downlink_queued` and `downlink_rejected` |

Validation rules:
- Exactly one of `data` and `entries` is present.
- Every `data` value is valid base64 and decodes to at most 200 bytes, the radio downlink payload limit.
- Every entry has a `packetCnt` (0 to 4294967295), and no counter repeats.
- Raw MQTT payload maximum: 1 MB.
- A refused command is reported on `event/downlink_rejected`; nothing is queued.

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
3. When a base station sends the downlink, or it expires, you receive `.../event/downlink_result` with the same `queId`.

### Observe Attach/Detach Lifecycle

1. Subscribe to a single device:
   ```bash
   mosquitto_sub -h localhost -p 1883 \
     -u admin -P KiloCenter \
     -t "mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/+" -v
   ```
2. When the endpoint attaches or detaches, you receive `.../event/attach` or `.../event/detach`.

## QoS Behavior

- `event/up` uses QoS 1 (at least once delivery).
- Lifecycle events (`attach`, `detach`, `downlink_queued`, `downlink_rejected`, `downlink_result`) use the configured events QoS.
- `command/down` subscriptions use the configured downlink QoS.

Your client should be idempotent because QoS 1 may deliver duplicate messages.

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
