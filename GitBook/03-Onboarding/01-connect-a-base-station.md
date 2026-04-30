# Connect a Base Station

## Goal

Connect a MIOTY base station to your local KiloCenter instance and confirm it is online.

## Prerequisites

- KC-Core, KC-Gateway, and KC-Web running (see [Installation](../02-GettingStarted/03-installation-docker-compose.md))
- TLS certificates generated (see the certificate bootstrap step in your installation guide)
- BSSCI reachable on port `5000`
- Server CA certificate available at `kilocenter-modules/KC-Core/certificates/ca.crt`

## Step 1: Create the Base Station in KC-Web

Before the hardware can connect, the base station must exist in KiloCenter.

1. Open KC-Web at `http://localhost/` (container) or `http://localhost:5173` (source dev).
2. Navigate to **Base Stations** in the sidebar.
3. Click **Add Base Station**.
4. Enter the base station **EUI** (16-character hex string matching the hardware identity) and a **name**.
5. Save the base station.

## Step 2: Configure Base Station Hardware

Point the base station firmware to your KiloCenter host:

- **Service Center address**: your KC-Core host IP or hostname
- **Service Center port**: `5000` (BSSCI)
- **Protocol**: TLS (required)

For AVA base stations, the factory login is `ubuntu` / `123456`. Change these credentials after initial access.

## Step 3: Import CA Certificate

The base station must trust KiloCenter's CA certificate for TLS to succeed.

1. Copy `kilocenter-modules/KC-Core/certificates/ca.crt` to the base station.
2. Import it into the base station's TLS trust store through its configuration interface.

## Step 4: Verify Connection

### In KC-Web

Navigate to **Base Stations**. The connected base station should show an **Online** status.

### In Logs

```bash
tail -f kilocenter-modules/logs/runtime/kc-core.log
```

A successful connection produces log entries showing:

- **Version negotiation** -- KC-Core and the base station agree on BSSCI protocol version
- **Attach operations** -- the base station registers its endpoints with KC-Core

Look for entries containing `versionNeg` and `attach` with no error messages.

### Common Connection Failures

| Symptom | Likely Cause |
|---------|-------------|
| No connection attempt in logs | Base station not pointing to correct host/port |
| TLS handshake error | CA certificate not imported or certificate mismatch |
| Connection drops immediately | Base station EUI not registered in KC-Web |
| Version negotiation fails | Firmware version incompatible with KC-Core BSSCI implementation |
