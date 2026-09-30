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
3. Click **Add End Point**.
4. Fill in the required fields:
   - **EUI**: the 16-character hex EUI (e.g., `0011223344556677`)
   - **Name**: a descriptive label
   - **Network Session Key**: the 32-character hex key
   - **Short Address**: the assigned short address
5. If the endpoint supports bidirectional communication, enable the **Bidirectional** flag.
6. To attach the endpoint right away, enable **Pre-Attachment**. Unidirectional endpoints need it, because they cannot attach over the air.
7. Save the endpoint.

## Validate Endpoint State

A new endpoint appears in the **Endpoints** list as detached. It becomes attached when you create it with **Pre-Attachment**, when you attach it (the endpoint's **Attach** action or the `AttachEndPoint` API), when an application center registers it with `preAttach`, or when the endpoint attaches over the air.

KiloCenter sends an attached endpoint to every connected base station and to every base station that connects later. An endpoint you detach, one an application center deregisters, and one that detaches over the air is detached at every connected base station. A base station that was away and resumes its session is sent the detachment when it reconnects; one that starts a new session is not sent the endpoint at all.

## How Do I Decode an Endpoint's Uplinks with a Blueprint?

1. Under **Blueprints**, upload a blueprint for the device's manufacturer and model.
2. Open the endpoint and click **Assign Blueprint** under **Blueprint Configuration** (or the **Edit** icon).
3. In **Blueprint Configuration**, pick the manufacturer and the model, then click **Save Changes**.

The model's default blueprint decodes every uplink received from then on; earlier uplinks keep the decode status they were stored with. Each uplink in the endpoint's **Activity** tab shows its decoded values, or why it was not decoded, below its payload.

## Optional MIOTY Profile Fields (Enterprise Edition)

The MIOTY specification defines additional endpoint profile fields that control radio behavior. These fields are available in the Enterprise Edition:

| Field | Type | Description |
|-------|------|-------------|
| `dualChan` | Boolean | Dual channel mode |
| `repetition` | Boolean | Downlink repetition |
| `wideCarrOff` | Boolean | Wide carrier offset |
| `longBlkDist` | Boolean | Long downlink interblock distance |

In the Community Edition, these fields use default values and are managed automatically by KC-Core during the attach process.

## Carrier Offset (Device Metadata)

The **Carrier Offset (informational, Hz)** field under **Device Metadata** records a measured or datasheet carrier offset for your own reference. It is a whole number of Hz between -2147483648 and 2147483647, it is never sent to a base station, and it has no effect on reception. Clearing the field in **Edit** removes the recorded value.

The carrier offset range a base station tolerates is set by the `wideCarrOff` profile flag above (radio protocol specification §3.5.4), not by this field.

## Common Failures

| Symptom | Likely Cause |
|---------|-------------|
| "Invalid EUI" error | EUI is not exactly 16 hex characters |
| "Duplicate endpoint" error | An endpoint with this EUI already exists |
| Endpoint stays detached | It was created without Pre-Attachment and has not been attached or attached over the air |
| Uplinks not appearing | Network session key mismatch between endpoint hardware and KiloCenter |
