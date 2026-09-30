# Configuration Basics

## Goal

Understand and verify baseline runtime configuration for a local deployment.

## Configuration Files

- `config/config.docker.yaml` -- KC-Core configuration (container mode)
- `KC-Core/config.yaml` -- KC-Core configuration (source dev mode)
- `KC-Gateway/config.yaml` -- KC-Gateway configuration (source dev mode)
- `docker-compose.yml` -- service definitions and port mappings

## Container Mode Defaults

In container mode, KC-Core reads `config/config.docker.yaml`. All service addresses use Docker service DNS names (e.g., `postgres`, `redis`, `mosquitto`). Environment variables in `.env` override the compose defaults.

## Source Dev Mode Defaults

`start-dev.sh` exports environment variables and auto-detects the PostgreSQL port:

- If Docker container `kilocenter-postgres` is running, it uses port `5433`.
- Otherwise, it falls back to `5432` for a host-local PostgreSQL.

## Settings to Verify Before Testing

- **Database**: host, port, credentials (default: `localhost:5433`, user `kilocenter`, password `changeme`)
- **KC-Core health**: port `8086`
- **KC-Gateway gRPC-web**: port `9090` (the documented browser ingress; KC-Core's own gRPC-web multiplexing is off by default and is enabled with `grpc.web.enabled: true` only when a deployment talks to KC-Core directly)
- **KC-Gateway health**: port `8087`
- **BSSCI**: port `5000` in `config/config.docker.yaml`, `5005` in `KC-Core/config.yaml`, TLS certificate paths
- **SCACI**: port `5001`, TLS certificate paths
- **MQTT**: broker host, port, and credentials
- **Dashboard service status**: the rows follow the `general`, `protocol`, `grpc`, `identity` and `mqtt` settings automatically; `status.timeout` is the only status setting

## What Is the Platform Tenant?

`general.tenant_id` (default `1`) names the platform tenant. KiloCenter files
operator and audit events that belong to the whole installation under it:
users being created, changed or deleted, organizations and API keys being
deleted, and server certificates being generated or renewed. Every edition
requires a value greater than zero and refuses to start without one.

Treat the platform tenant as reserved for these events. Only administrators
can read them, and members of an organization that shares the tenant see only
the event categories their roles cover. See
[User Roles and Permissions](../05-Security/04-users-and-roles.md).

## TLS Certificate Paths

BSSCI and SCACI TLS certificate paths are configured under the `protocol` section in `KC-Core/config.yaml` (source dev) or `config/config.docker.yaml` (container):

```yaml
protocol:
  bsci_tls:
    enabled: true
    cert_file: "certificates/server.crt"
    key_file: "certificates/server.key"
    ca_file: "certificates/ca.crt"
    min_version: "1.2"
  scaci_tls:
    enabled: true
    cert_file: "certificates/server.crt"
    key_file: "certificates/server.key"
    ca_file: "certificates/ca.crt"
    min_version: "1.3"
```

Paths in the source dev config are relative to the KC-Core working directory. Container config uses absolute paths (`/app/certificates/`).

- **Docker Compose:** Server certificates are generated automatically on first `docker compose up`. See [Docker Compose Installation](03-installation-docker-compose.md).
- **Linux Host:** Generate server certificates manually with `KC-Core/certgen` before starting KC-Core. See [Linux Host Installation](04-installation-linux-host.md).

## Downlink Lifetime

A downlink waits in the service center queue until the endpoint opens a downlink window. `protocol.downlink_expiry` bounds that wait:

```yaml
protocol:
  downlink_expiry:
    lifetime: "24h"        # how long a queued downlink waits for a window
    sweep_interval: "5s"   # how often overdue downlinks are expired
    batch_size: 100        # downlinks expired per statement
```

A downlink older than `lifetime` is never sent. It is marked expired and reported as `expired` to the Application Center that queued it (`dlDataRes`), on the MQTT `downlink_result` topic, and in the downlink results. In the Helm chart the keys are `kcCore.config.protocol.downlinkExpiry.lifetime`, `sweepInterval` and `batchSize`.

## Application Center Session Resumption

When an Application Center loses its SCACI connection, its session stays resumable. Every uplink (`ulData`), endpoint status (`epStat`) and result of a downlink it queued (`dlDataRes`) produced while it is disconnected is kept for that session and sent when the Application Center resumes it (`snResume` true), before anything new, with operation IDs that continue from before the loss. This also holds across a KC-Core restart. An Application Center that starts a new session instead receives none of it.

An Application Center is its `acEui` within its organization. Application Centers of two organizations that use the same `acEui` each keep their own session: neither replaces or resumes the other's, and each receives only the results of the downlinks it queued.

`protocol.scaci_resume_max_pending_operations` bounds what a disconnected session holds:

```yaml
protocol:
  scaci_resume_max_pending_operations: 10000
```

When a session already holds that many operations, the next one ends its resumability: the service center discards the session, and the Application Center starts a new session when it reconnects. Uplinks and downlink results remain available through MQTT and the API. In the Helm chart the key is `kcCore.config.scaci.resumeMaxPendingOperations`.

## Default Credentials

These defaults exist for local development convenience. Change them for any non-local deployment:

- PostgreSQL: user `kilocenter`, password `changeme`
- MQTT broker: user `admin`, password `KiloCenter`
- AVA base station factory login: `ubuntu` / `123456`

## MQTT Integration

MQTT is **disabled by default** in `KC-Core/config.yaml`:

```yaml
mqtt:
  enabled: false
  host: "localhost"
  port: 1883
  username: "admin"
  password: "KiloCenter"
```

To enable MQTT, set `enabled: true` and restart KC-Core. The Mosquitto broker must be running (it is included in the Docker Compose stack). See [MQTT First Steps](../04-Integrations/03-mqtt-first-steps.md) for integration details.

## Validation Checklist

- KC-Core starts without errors and `/health` responds on port `8086`
- KC-Identity starts and `/health` responds on port `8088`
- KC-Gateway starts and `/health` responds on port `8087`
- **Container mode**: KC-Web loads at `http://localhost/`
- **Source dev mode**: KC-Web loads at `http://localhost:5173`
- KC-Web can reach KC-Gateway on port `9090` (gRPC-web requests succeed in browser console)
