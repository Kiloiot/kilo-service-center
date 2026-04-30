# Verify the First Uplink

## Goal

Confirm that uplink data travels end-to-end from an endpoint through a base station to KiloCenter.

## Prerequisites

- At least one base station connected and online
- At least one endpoint registered with matching keys (see [Register an Endpoint](02-register-an-endpoint.md))
- The endpoint is transmitting data

## Check Uplinks in KC-Web

1. Open KC-Web at `http://localhost/` (container) or `http://localhost:5173` (source dev).
2. Navigate to **Endpoints** and select your endpoint.
3. Open the **Messages** tab.
4. Look for uplink entries showing received payload data, timestamp, RSSI, and SNR values.

If messages appear here, your uplink path is working end-to-end.

## Check KC-Core Logs

```bash
tail -f kilocenter-modules/logs/runtime/kc-core.log
```

Look for `ulData` entries that include your endpoint's EUI. These log lines confirm KC-Core received and processed the uplink from the base station.

## Programmatic Verification (Optional)

You can also verify uplinks through the gRPC API:

```bash
grpcurl -plaintext -d '{"endpoint_eui": "<EP_EUI_HEX>"}' \
  localhost:9090 kilocenter.api.v1.KiloCenterService/ListMessages
```

## Troubleshooting

| Symptom | Likely Cause |
|---------|-------------|
| No messages in KC-Web | Endpoint not transmitting, or not attached by any base station |
| Messages appear in logs but not in KC-Web | Browser cache or KC-Web not connected to KC-Gateway |
| `ulData` errors in logs | Network session key mismatch or endpoint not registered |
| RSSI/SNR values missing | Base station firmware not reporting radio metadata |
