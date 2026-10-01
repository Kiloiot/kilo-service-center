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
2. Navigate to **Endpoints**, select your bidirectional endpoint and open the **Downlink** tab.
3. Compose the downlink:
   - **Payload (hex)**: the user data as hex (e.g. `01020304`); leave it empty for an acknowledgement-only downlink
   - **Format** and **Priority** (`prio`), and under **Advanced Options** the `responseExp`, `responsePrio`, `dlWindReq`, `expOnly` and `dlRxStatQry` flags
4. Select **Send Downlink**. The downlink appears under the endpoint's **Traffic** tab > **Downlink Queue**; while no base station has taken it yet you can edit or revoke it there.

## Send a Downlink via gRPC (Alternative)

In the JSON form of a gRPC request, `payloads` entries are base64 (`AQIDBA==` is bytes `01 02 03 04`).

```bash
grpcurl -plaintext \
  -H "authorization: Bearer <JWT_TOKEN>" \
  -H "x-organization-id: <ORG_ID>" \
  -d '{
  "epEui": "<EP_EUI_HEX>",
  "payloads": ["AQIDBA=="],
  "priority": 0.5
}' localhost:9090 kilocenter.api.v1.KiloCenterService/SendDownlink
```

## Verify the Result

### In KC-Web

Open **Traffic** (in the navigation, or the endpoint's **Traffic** tab) and select **Downlink Results**. A result shows `result` (`sent`, `expired` or `invalid`), and for a sent downlink the transmitting `bsEui`, `txTime` and `packetCnt`; `dlAck` shows when the endpoint acknowledged it in its next uplink. **Downlink Queue** shows the downlinks still waiting for a transmission window.

### In Logs

```bash
tail -f kilocenter-modules/logs/runtime/kc-core.log
```

Look for `dlDataQue`, `dlDataQueRsp`, and `dlDataQueCmp` entries -- these represent the three-way handshake of the downlink operation.

## Troubleshooting

| Symptom | Likely Cause |
|---------|-------------|
| Downlink stays queued | The base station that last heard the endpoint is offline or not bidirectional; the downlink goes out in the endpoint's next downlink window |
| Downlink refused as not bidirectional | The endpoint is registered without the `bidi` flag |
| Downlink reported `expired` | No downlink window opened within `protocol.downlink_expiry.lifetime` (see [Configuration Basics](../02-GettingStarted/06-configuration-basics.md)) |
| Downlink stays **Revoking** in the queue | A base station held it when its deadline passed; it ends when the station answers the revoke or reconnects. If the station refuses revokes with a code other than `protocol.downlink_expiry.revoke_not_held_codes`, see [Configuration Basics](../02-GettingStarted/06-configuration-basics.md) |
| No result returned | Base station did not complete the downlink handshake |
| Payload rejected | Data not valid base64 or exceeds maximum payload size |
