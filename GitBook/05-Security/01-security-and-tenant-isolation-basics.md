# Security Basics

## Goal

Establish a secure baseline for KiloCenter deployments.

## Default Credentials

Local development includes convenience credentials that must be changed for any shared or production environment:

| Service | Default | Notes |
|---------|---------|-------|
| PostgreSQL | user `kilocenter`, password `changeme` | Change in `config.yaml` and Docker Compose |
| MQTT broker | user `admin`, password `KiloCenter` | Change in Mosquitto config and `config.yaml` |

## TLS for Base Station and Application Center Communication

BSSCI requires TLS 1.2 or higher. SCACI requires TLS 1.3 or higher. Every connection to KC-Core uses TLS encryption.

### CA Trust Model

KiloCenter uses a self-signed Certificate Authority (CA) to issue all certificates:

| Certificate | Purpose | Default Validity | Location |
|-------------|---------|-----------------|----------|
| CA certificate | Root of trust; distributed to base stations | 20 years | `certificates/ca.crt` |
| CA private key | Signs server and client certificates | -- | `certificates/ca.key` |
| Server certificate | KC-Core BSSCI/SCACI TLS listeners | 1 year | `certificates/server.crt` |
| Server private key | TLS handshake | -- | `certificates/server.key` |
| Client certificate | Per-base-station mutual TLS (optional) | 1 year | Generated on demand |

Base stations trust the **CA certificate**, not individual server certificates. This means server certificates can be renewed without touching base stations, as long as the same CA signs them.

If you regenerate the CA, all existing server and client certificates become invalid and must be reissued. Back up `ca.key` securely.

### Generating Certificates

#### Docker Compose Deployments

Server certificates are generated automatically on first `docker compose up`.
To use a custom hostname, set `KILOCENTER_TLS_SERVER_NAME` in `.env` before the first start.

To regenerate (e.g., after hostname change): `docker compose down -v` and start again.
**Warning:** `down -v` removes all Docker volumes including the database.

#### Linux-Host Deployments

Build the certificate generator and create certificates before starting KC-Core:

```bash
go build -o KC-Core/certgen KC-Core/cmd/certgen/main.go
KC-Core/certgen -dir KC-Core/certificates -days 365 -server your-hostname.example.com
```

Renew server certificate only (CA already exists):

```bash
KC-Core/certgen -dir KC-Core/certificates -days 365 -server your-hostname.example.com -server-only
```

#### Client Certificate Generation

**Docker Compose:**

```bash
docker compose run --rm certgen \
    -dir /app/certificates -client-only -client 70-B3-D5-9C-D0-00-09-E6
```

**Linux Host:**

```bash
KC-Core/certgen -dir KC-Core/certificates -client-only -client 70-B3-D5-9C-D0-00-09-E6
```

#### certgen Reference

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `certs` | Output directory for certificate files |
| `-server` | `localhost` | Server hostname (used as CN and SAN) |
| `-days` | `365` | Server/client certificate validity in days |
| `-ca-years` | `20` | CA certificate validity in years |
| `-ca-only` | `false` | Generate only the CA certificate |
| `-server-only` | `false` | Generate only the server certificate (CA must already exist) |
| `-client-only` | `false` | Generate only a client certificate (CA must already exist) |
| `-client` | (empty) | Client name for client certificate (e.g., base station EUI) |

### Certificate Rotation

**Docker Compose:** To regenerate with a new hostname, run `docker compose down -v` and start again. **Warning:** `down -v` removes all volumes including the database.

**Linux Host:** Renew the server certificate:

```bash
KC-Core/certgen -dir KC-Core/certificates -days 365 -server your-hostname.example.com -server-only
```

Then restart KC-Core.

**Via GUI** (post-install renewal):

1. Open KC-Web and navigate to **Certificates**.
2. Click **Renew Server Certificates** and confirm.
3. Restart KC-Core to load the new certificates.

**After rotation:**

1. Restart KC-Core to load the new certificates.
2. Verify base station reconnections succeed.
3. If the CA was changed, redistribute `ca.crt` to all base stations.

### Certificate Configuration

KC-Core loads certificates from paths configured in `config.yaml` (source dev) or `config/config.docker.yaml` (container):

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

Paths in source dev mode are relative to the KC-Core working directory. KC-Core will fail to start if these files are missing.

## Network Exposure

Limit which ports are accessible from outside your local network:

| Port | Service | Exposure Recommendation |
|------|---------|------------------------|
| 5000 | BSSCI | Only base station networks |
| 5001 | SCACI | Only application center hosts |
| 9090 | KC-Gateway (gRPC-web) | Operator and API consumer networks |
| 80   | KC-Web (container) | Operator networks only |
| 50051 | KC-Core internal gRPC | Loopback only, never expose externally |
| 5433 | PostgreSQL | Loopback only |
| 6379 | Redis | Loopback only |
| 1883 | MQTT | Only MQTT consumer networks |

## Hardening Checklist

- [ ] Rotate all default credentials listed above
- [ ] Restrict network access to management ports (50051, 5433, 6379)
- [ ] Store certificate private keys with restricted file permissions
- [ ] Enable audit-level logging in production environments
- [ ] Review `config.yaml` for any remaining development defaults

## Enterprise Edition

The Enterprise Edition adds multi-tenant security features:

- User authentication with JWT validation
- Organization-scoped data isolation
- Role-based access control

These features are not available in the Community Edition.
