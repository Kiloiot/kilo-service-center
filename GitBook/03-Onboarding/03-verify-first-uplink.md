# Verify the First Uplink

## Goal

Confirm that uplink data travels end-to-end from an endpoint through a base station to KiloCenter.

## Prerequisites

- At least one base station connected and online
- At least one endpoint registered with matching keys (see [Register an Endpoint](02-register-an-endpoint.md))
- The endpoint is transmitting data

## Check Uplinks in KC-Web

1. Open KC-Web at `http://localhost/` (container) or `http://localhost:5173` (source dev).
2. Navigate to **Endpoints** and select your endpoint. Its **Activity** tab lists the endpoint's events and uplinks in one table, newest first, with the same columns as the Events Log: each uplink is an `ulData` row showing its `opId`, the base stations that received it, and "Uplink #" with the `packetCnt` over its `rssi`, `snr`, `eqSnr`, the `dlOpen`/`dlAck` flags and the payload size. Expand an uplink for every base station's reception and its subpackets, the `ulData` message and, when a blueprint decoded it, the decoded payload. A base station's **Activity** tab shows the same for the uplinks that station heard, each named by its endpoint, and exports them as CSV or JSON.
3. Open the **Traffic** tab. **Uplink Events** lists each uplink with `rxTime`, the receiving `bsEui`, `packetCnt`, `snr`, `rssi`, `eqSnr`, the `dlOpen`, `responseExp`, `dlAck` and `duplicate` flags and `userData` in hex; expand a row for every base station's reception and its subpackets.
4. To see the uplinks of every endpoint at once, open **Traffic** in the navigation.

If uplinks appear here, your uplink path is working end-to-end. New uplinks appear without a page reload.

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
