# Register an Endpoint

## Goal

Create an endpoint in KiloCenter so it can be attached by a base station and exchange traffic.

## Prerequisites

- KC-Core, KC-Gateway, and KC-Web running
- At least one base station connected (see [Connect a Base Station](01-connect-a-base-station.md))

## Required Fields

| Field | Format | Description |
|-------|--------|-------------|
| `epEui` | 16-character hex string (8 bytes) | Unique endpoint identifier. Must match the hardware EUI. |
| `name` | Free text | Human-readable label for the endpoint. |
| `nwkSnKey` | 32-character hex string (16 bytes) | Network session key for frame authentication. |
| `shAddr` | Integer | Short address assigned to the endpoint. |

These values come from the endpoint provisioning process. The EUI and network session key must match what is programmed into the endpoint hardware.

## Register Through KC-Web

1. Open KC-Web at `http://localhost/` (container) or `http://localhost:5173` (source dev).
2. Navigate to **Endpoints** in the sidebar.
3. Click **Add Endpoint**.
4. Fill in the required fields:
   - **EUI**: the 16-character hex EUI (e.g., `0011223344556677`)
   - **Name**: a descriptive label
   - **Network Session Key**: the 32-character hex key
   - **Short Address**: the assigned short address
5. If the endpoint supports bidirectional communication, enable the **Bidirectional** flag.
6. Save the endpoint.

## Validate Endpoint State

After registration, the endpoint should appear in the **Endpoints** list with a registered status. Once a base station performs an attach operation for this endpoint, the status will update to reflect the active connection.

## Optional MIOTY Profile Fields (Enterprise Edition)

The MIOTY specification defines additional endpoint profile fields that control radio behavior. These fields are available in the Enterprise Edition:

| Field | Type | Description |
|-------|------|-------------|
| `dualChan` | Boolean | Dual channel mode |
| `repetition` | Boolean | Downlink repetition |
| `wideCarrOff` | Boolean | Wide carrier offset |
| `longBlkDist` | Boolean | Long downlink interblock distance |

In the Community Edition, these fields use default values and are managed automatically by KC-Core during the attach process.

## Common Failures

| Symptom | Likely Cause |
|---------|-------------|
| "Invalid EUI" error | EUI is not exactly 16 hex characters |
| "Duplicate endpoint" error | An endpoint with this EUI already exists |
| Endpoint stays in registered state | Base station has not yet attached this endpoint |
| Uplinks not appearing | Network session key mismatch between endpoint hardware and KiloCenter |
