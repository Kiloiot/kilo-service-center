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
- **KC-Gateway gRPC-web**: port `9090`
- **KC-Gateway health**: port `8087`
- **BSSCI**: port `5000`, TLS certificate paths
- **SCACI**: port `5001`, TLS certificate paths
- **MQTT**: broker host, port, and credentials

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
