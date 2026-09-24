# Installation: Docker Compose

## Before starting an installation

Use these instructions only in an isolated evaluation environment with test data. The current
examples contain published credentials and signing-key values, and some ports are reachable beyond
the host unless your network blocks them. Changing only the administrator password does not correct
all of these defaults. Do not expose the example installation to the Internet or use it for customer
data. Read the [installation safety notice](../05-Security/02-installation-safety.md) before running commands.


## Goal

Start a working local stack using Docker Compose. The full stack — including KC-Web — runs entirely in containers. No host Go toolchain or Bun runtime is required.

## Architecture

The `docker compose up` command starts four application services and three infrastructure services. A one-shot `certgen` init container generates TLS certificates on first boot. Services start in dependency order: PostgreSQL, Redis, and Mosquitto first, then `certgen`, then KC-Identity, KC-Core, KC-Gateway, and finally KC-Web.

| Service | Container | Port(s) | Description |
|---------|-----------|---------|-------------|
| `postgres` | kilocenter-postgres | 5433→5432 | PostgreSQL database |
| `redis` | kilocenter-redis | 6379 | Redis cache |
| `mosquitto` | kilocenter-mosquitto | 1883, 9001 (WebSocket) | MQTT broker |
| `certgen` | kilocenter-certgen | — | One-shot TLS certificate generator |
| `kc-identity` | kilocenter-identity | 50052 (gRPC, internal), 8088 (health) | Identity, users, organizations |
| `kilocenter` | kilocenter-app | 50051 (gRPC, internal), 5000 (BSSCI), 5001 (SCACI), 8086 (health) | KC-Core service center engine |
| `kc-gateway` | kilocenter-gateway | 9090 (gRPC-web), 8087 (health) | External API ingress |
| `kc-web` | kilocenter-web | 80 | Web management interface (nginx) |

Internal gRPC ports (50051, 50052) stay on the Docker network and are not published to the host by default.

## Step 1: Configure Environment

From the repository root:

```bash
cp .env.example .env
```

Edit `.env` if you want to change database credentials or log level. The defaults work for a local evaluation.

## Step 2: TLS Certificates (Automatic)

Certificates are generated automatically during `docker compose up`. The `certgen`
service creates a self-signed CA and server certificate (valid 365 days, hostname
`localhost`) if none exist in the `cert_data` volume.

**Custom hostname** — set `KILOCENTER_TLS_SERVER_NAME` in `.env` **before the first start**:

```bash
echo 'KILOCENTER_TLS_SERVER_NAME=bssci.example.com' >> .env
```

Set the hostname before first start. For a later change, use
[certificate-only renewal](../05-Security/03-certificate-renewal.md). Do not remove Docker volumes
to renew a certificate: that would also delete the database and existing trust keys.

See [Security](../05-Security/01-security-and-tenant-isolation-basics.md) for client
certificate generation.

## Step 3: Start All Services

```bash
docker compose up --build -d
```

This builds and starts all services. Subsequent starts after a rebuild use cached layers where possible.

## Step 4: Validate Startup

```bash
# Check all services are up
docker compose ps

# Export CA certificate for base stations
docker compose cp kilocenter:/app/certificates/ca.crt ./ca.crt

# KC-Web UI
curl -s -o /dev/null -w "%{http_code}" http://localhost/    # 200

# KC-Core health
curl -s http://localhost:8086/health

# KC-Identity health
curl -s http://localhost:8088/health

# KC-Gateway health
curl -s http://localhost:8087/health

# Verify nginx gRPC-web proxy (200/401/403 = OK; 502/404 = proxy broken)
curl -s -o /dev/null -w "%{http_code}" \
  -X POST \
  -H "Content-Type: application/grpc-web+proto" \
  -H "X-Grpc-Web: 1" \
  --data-binary $'\x00\x00\x00\x00\x00' \
  http://localhost/kilocenter.api.v1.KiloCenterService/GetSystemStatus
```

Then open KC-Web in your browser:

- [http://localhost/](http://localhost/)

## Default Admin Account

A default admin user is created on first startup via database migration:

| | |
|---|---|
| **Email** | `admin@kilocenter.local` |
| **Password** | `admin123!` |

This account has full admin privileges including tenant, base station, and endpoint management.

The account is for isolated evaluation. A production setup needs installation-owned credentials,
protected signing keys and verified network restrictions, not just an administrator-password change.
Use the normal user-management interface to manage accounts. Do not edit identity tables by hand
as a troubleshooting shortcut; doing so can leave related sessions or permissions inconsistent.

> **Note:** MQTT integration is disabled by default. The Mosquitto broker runs in Docker but KC-Core does not connect to it until you enable MQTT in `config/config.docker.yaml`. See [MQTT First Steps](../04-Integrations/03-mqtt-first-steps.md) when you are ready to set up MQTT.

## MQTT Broker Credentials

The Mosquitto broker runs with password authentication. Default credentials (change for production):

| Username | Password | Purpose |
|----------|----------|---------|
| `admin` | `KiloCenter` | Administration |
| `kilocenter` | `kilocenter123` | KC-Core service account |

## Common Operations

```bash
# View logs for a service
docker compose logs -f kilocenter
docker compose logs -f kc-web

# Restart a single service
docker compose restart kilocenter

# Stop all services
docker compose down

# Stop and remove volumes (destructive — deletes database)
docker compose down -v
```

## Volumes

| Volume | Contents |
|--------|----------|
| `postgres_data` | PostgreSQL data files |
| `redis_data` | Redis persistence |
| `mosquitto_data` | MQTT retained messages |
| `mosquitto_log` | MQTT logs |
| `cert_data` | TLS certificates (auto-generated on first boot) |

## Authentication Secret (HMAC)

KC-Identity signs JWT tokens and KC-Gateway validates them. Both services must use the **same HMAC secret**. The default development secret is set in two config files:

- `config/config.identity-docker.yaml` → `auth.hmac_secret`
- `config/config.gateway-docker.yaml` → `auth.hmac_secret`

If these values differ, login will succeed but every subsequent API call will fail with `invalid_token`. When changing the secret for production, update **both files** to the same value (minimum 32 characters).

For `invalid_token`, first check that the identity and gateway use the intended matching key and
that your browser is connected to the correct installation. Sign out and sign in again. If the
problem remains, retain the error code and time and use [Support](../../SUPPORT.md). Do not clear
all Redis data or delete every user's refresh tokens to diagnose one login failure. A confirmed
key compromise needs a planned key rotation and session-revocation procedure.

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `certgen` writes files owned by root | UID/GID mismatch | Set `UID=$(id -u)` and `GID=$(id -g)` in `.env` |
| `kc-web` shows gRPC errors | `kc-gateway` not healthy | Check `docker compose ps` and gateway logs |
| `kilocenter` exits immediately | Missing certificates | Check `certgen` logs: `docker compose logs certgen` |
| Port 80 already in use | Another web server running | Stop it or change the host port mapping |
| Base station TLS failure | Certificate mismatch | Ensure base station trusts the CA certificate (`ca.crt`) |
| `invalid_token` after login | HMAC secret mismatch | Ensure `hmac_secret` is identical in `config.identity-docker.yaml` and `config.gateway-docker.yaml` |

