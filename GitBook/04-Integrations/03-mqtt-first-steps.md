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

## Observe Downlink Results

```bash
mosquitto_sub -h localhost -p 1883 \
  -u admin -P KiloCenter \
  -t 'mioty/<ORG_UUID>/device/<EP_EUI_HEX>/event/downlink_result' -v
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
  "data": "SGVsbG8="
}
```

- `data` is base64-encoded payload bytes.
- `rxTime` is a Unix timestamp in nanoseconds.

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

### `event/downlink_result`

```json
{
  "epEui": "70b3d59cd00009e6",
  "queId": 12345,
  "result": "sent"
}
```

Possible `result` values: `sent`, `expired`, `invalid`.

### `command/down` (You Publish This)

```json
{
  "data": "AQIDBA==",
  "confirmed": false
}
```

Validation rules:
- `data` must be valid base64.
- Decoded payload must be non-empty.
- Raw MQTT payload maximum: 1 MB.
- Decoded `data` maximum: 256 KB.

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
2. KiloCenter resolves the org, queues the downlink, and processes it.
3. After processing, you receive a message on `.../event/downlink_result`.

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
- Lifecycle events (`attach`, `detach`, `downlink_result`) use the configured events QoS.
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
2. Verify the topic is exactly `mioty/<ORG_UUID>/device/<EP_EUI_HEX>/command/down`.
3. Ensure the payload JSON is valid and `data` is valid base64.
4. Check that ACL allows write access to your org's command topic.

### Quick Broker Sanity Check

Subscribe to all topics to verify the broker is receiving messages:

```bash
mosquitto_sub -h localhost -p 1883 -u admin -P KiloCenter -t 'mioty/#' -v
```
