# Verify the First Downlink

## Goal

Queue a downlink command to an endpoint and verify the result.

## Prerequisites

- Endpoint registered with **bidirectional** flag enabled
- Base station connected and supporting bidirectional communication
- At least one successful uplink confirmed (see [Verify First Uplink](03-verify-first-uplink.md))

Both the endpoint and the base station must have bidirectional capability for downlinks to work.

## Send a Downlink via KC-Web

1. Open KC-Web at `http://localhost/` (container) or `http://localhost:5173` (source dev).
2. Navigate to **Endpoints** and select your bidirectional endpoint.
3. Use the downlink interface to queue a message:
   - **Payload**: base64-encoded data (e.g., `AQIDBA==` for bytes `01 02 03 04`)
   - **Confirmed**: whether to request delivery confirmation
4. Submit the downlink.

## Send a Downlink via gRPC (Alternative)

```bash
grpcurl -plaintext -d '{
  "endpoint_eui": "<EP_EUI_HEX>",
  "data": "AQIDBA==",
  "confirmed": false
}' localhost:9090 kilocenter.api.v1.KiloCenterService/SendDownlink
```

## Verify the Result

### In KC-Web

Check the endpoint's downlink results view for status updates. A successful downlink shows the delivery status and any confirmation from the endpoint.

### In Logs

```bash
tail -f kilocenter-modules/logs/runtime/kc-core.log
```

Look for `dlDataQue`, `dlDataQueRsp`, and `dlDataQueCmp` entries -- these represent the three-way handshake of the downlink operation.

## Troubleshooting

| Symptom | Likely Cause |
|---------|-------------|
| Downlink stays queued | Endpoint not attached or not bidirectional |
| "Not bidirectional" error | Endpoint or base station `bidi` flag not set |
| No result returned | Base station did not complete the downlink handshake |
| Payload rejected | Data not valid base64 or exceeds maximum payload size |
