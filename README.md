# KiloCenter - Open Source MIOTY Network Server

[![Build](https://img.shields.io/github/actions/workflow/status/Kiloiot/kilo-service-center/pipeline-build.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=build)](https://github.com/Kiloiot/kilo-service-center/actions/workflows/pipeline-build.yml)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Kiloiot/kilo-service-center?filename=KC-Core%2Fgo.mod&style=flat-square&logo=go&logoColor=white&label=go)](KC-Core/go.mod)
[![React](https://img.shields.io/github/package-json/dependency-version/Kiloiot/kilo-service-center/react?filename=KC-Web%2Fpackage.json&style=flat-square&logo=react&logoColor=white&label=react)](KC-Web/package.json)
[![Docs](https://img.shields.io/badge/docs-kiloiot.io-blue?style=flat-square&logo=gitbook&logoColor=white)](https://docs.kiloiot.io/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Kiloiot/kilo-service-center/KC-Core?style=flat-square)](https://goreportcard.com/report/github.com/Kiloiot/kilo-service-center/KC-Core)

KiloCenter is an open source MIOTY network server for running your own MIOTY network — managing base stations, endpoints, uplink and downlink traffic, and application integrations from a self-hosted service center.

It is a MIOTY-native service center built specifically for the MIOTY stack from the ground up. It is not a fork, wrapper, or adaptation of another LPWAN network server.

MIOTY is an LPWAN protocol designed for interference resilience, long-range communication, and massive endpoint fleets. Unlike LoRaWAN, it uses telegram splitting to stay robust under heavy interference, and is increasingly adopted for industrial IoT deployments at scales where other LPWAN options hit capacity walls. KiloCenter implements the service-center side of that stack with Docker and Linux-host deployment paths, gRPC and MQTT integrations, and a web management console.

## Live Demo

![KiloCenter service center dashboard](GitBook/images/service-center-dashboard.jpg)

A demo environment is available at [https://servicecenter.kiloiot.io/](https://servicecenter.kiloiot.io/)

- **Username:** `demo@kiloiot.io`
- **Password:** `demo1234`

## Production Use Notice

The community edition is provided as-is without commercial support. Production use is at your own risk.

For supported production deployments, use Kilo Cloud or the on-premise Kilo IoT Platform (Contact Sales). The enterprise edition includes commercial support, SLA-backed operation, multi-tenancy, and optional MIOTY capabilities including BSSCI sub-channels, the Variable MAC (VM) attachment, and ReCon extension support.

## Features

### MIOTY Network Server
- **MIOTY BSSCI Protocol v1.0.0** - TLS-secured base station to service center communication
- **Base Station Management** - certificate-based authentication and status monitoring
- **Endpoint Management** - endpoint registry with attachment propagation
- **Message Processing** - uplink and downlink handling with deduplication
- **Multi-Base Station Operation** - shared coverage across base stations within a self-hosted deployment
- **Telegram Splitting Support** - native handling of MIOTY PHY behavior
- **Pattern Diversity** - multi-pattern transmission support for robustness
- **Security** - AES-128 network and application key handling

### Service Center Features
- **BSSCI Interface** - base station to service center protocol support
- **SCACI Interface** - service center to application center protocol support with TLS, session persistence, monitoring, and queue/status flows
- **Event System** - centralized event persistence and alerting
- **Certificate Management** - CA-based trust model with rotation and monitoring support
- **Database Partitioning** - monthly partitioned message storage
- **Archival System** - data lifecycle and retention support
- **MQTT Integration** - Mosquitto-backed application integration path
- **gRPC API** - gRPC and gRPC-web management APIs through KC-Gateway

### Management Interface
- **Web Dashboard** - React-based management console
- **Real-time Monitoring** - live status, traffic, and alert views
- **Base Station Registration** - GUI-based provisioning workflows
- **Endpoint Registration** - GUI endpoint management workflows
- **Certificate Workflows** - server certificate status, renewal, and base station certificate workflows
- **Message Viewer** - traffic views for different message types

### Platform Features
- **Docker Compose Deployment** - `postgres`, `redis`, `mosquitto`, `certgen`, `kc-identity`, `kilocenter`, `kc-gateway`, and `kc-web`
- **Database Migrations** - version-controlled schema changes
- **Structured Logging** - configurable logs and health endpoints
- **Cross-platform Development** - Linux, macOS, and Windows via WSL

## Architecture

KiloCenter uses a modular service layout:

```text
+-----------------+     +-----------------+
|   Browser       | --> |    KC-Web       | (:80 nginx / :5173 dev)
+-----------------+     +--------+--------+
                                 |
                                 | gRPC-web
                                 v
+-----------------+     +-----------------+          +-----------------+
| External        | --> |  KC-Gateway     | -------> |  KC-Identity    |
| Applications    | gRPC| (:9090 external |  public  | (:50052 internal|
| / Integrators   | /   |  gRPC +         |  RPCs    |  :8088 health)  |
|                 |gRPC-|  gRPC-web,      |          +--------+--------+
|                 | web |  :8087 health)  |                   ^
+-----------------+     +--------+--------+                   |
                                 |                            | internal identity RPCs
                                 | trusted internal gRPC      |
                                 v                            |
+-----------------+     +-----------------+                   |
|   Base Station  | --> |    KC-Core      | ------------------+
|   (MIOTY BS)    |BSSCI| (:50051 internal|   org / membership /
|   (:5000 TLS)   |     |  :5000 BSSCI    |   admin resolution
+-----------------+     |  :5001 SCACI    |
                        |  :8086 health)  | -------> +-----------------+
+-----------------+     +--------+--------+   SQL    |   PostgreSQL    |
| Application     | ------------>|                   |   via KC-DB     |
| Center          |    SCACI TLS |                   | (:5433 host map)|
|   (:5001 TLS)   |              |                   +-----------------+
+-----------------+              |
                                 +-----------------> +-----------------+
                                 |                   |      Redis      |
                                 |                   |     (:6379)     |
                                 |                   +-----------------+
                                 |
                                 +-----------------> +-----------------+
                                                     |   Mosquitto     |
                                                     |   via KC-MQTT   |
                                                     | (:1883 / :9001) |
                                                     +-----------------+
```

This is the runtime view. `KC-Gateway` is the external API ingress for the service center. It exposes native gRPC and gRPC-web on port `9090`, so both `KC-Web` and third-party applications integrate with KiloCenter through the same public API surface. `KC-Core` also uses internal RPCs to `KC-Identity` for organization, membership, and admin-resolution flows. In Docker Compose, a one-shot `certgen` service also runs before KC-Core on first startup to create the shared CA and server certificate set. Repository modules are listed below.

### Repository Modules

- **KC-Web** - web management interface served by nginx on port 80
- **KC-Gateway** - external gRPC-web ingress and request proxy on port 9090
- **KC-Identity** - identity and organization service on port 50052
- **KC-Core** - BSSCI, SCACI, internal gRPC API, and service-center logic
- **KC-DB** - PostgreSQL migrations, repositories, and storage interfaces used by the runtime services
- **KC-MQTT** - MQTT integration package used by KC-Core; not deployed as a separate container in the community Docker stack
- **pkg** - shared versioning and common Go utilities used across modules

## Getting Started

### Quick Start (Docker Compose)

Prerequisites:

- Docker and Docker Compose v2
- Git

1. Clone the repository:

```bash
git clone https://github.com/Kiloiot/kilo-service-center.git
cd KiloServiceCenter
```

2. Copy the environment template:

```bash
cp .env.example .env
```

3. Start the stack:

```bash
docker compose up --build -d
```

This starts `postgres`, `redis`, `mosquitto`, `certgen`, `kc-identity`, `kilocenter`, `kc-gateway`, and `kc-web`.

TLS certificates are generated automatically on first boot. Database migrations run on startup.

4. Open the web interface and log in with the default admin account:

```text
http://localhost/
```

| | |
|---|---|
| **Email** | `admin@kilocenter.local` |
| **Password** | `admin123!` |

> **Warning:** Change the default admin password or remove this account before any production or public-facing deployment. The default credentials are published in this repository.

> **Important:** KC-Identity and KC-Gateway must share the same HMAC secret for JWT authentication to work. The secret is configured in `config/config.identity-docker.yaml` and `config/config.gateway-docker.yaml` under `auth.hmac_secret`. If these values differ, you will see `invalid_token` errors after login. See the [Docker Compose guide](GitBook/02-GettingStarted/03-installation-docker-compose.md#authentication-secret-hmac) for details.

For Docker-specific detail, custom TLS hostname setup, and CA export, see [Docker Compose Installation](GitBook/02-GettingStarted/03-installation-docker-compose.md).

### Other Install Modes

- [Linux Host Installation](GitBook/02-GettingStarted/04-installation-linux-host.md) - manual host deployment with manual `KC-Core/certgen`
- [Local VM and Automation](GitBook/02-GettingStarted/05-installation-local-vm-and-automation.md) - Docker-based VM workflow
- [Getting Started](GitBook/02-GettingStarted/README.md) - architecture, prerequisites, and installation routing

### Exporting the CA Certificate

Base stations and external SCACI application centers need the CA certificate to establish TLS trust:

```bash
docker compose cp kilocenter:/app/certificates/ca.crt ./ca.crt
```

### Stopping

```bash
# Stop services and keep data volumes
docker compose down

# Full reset: removes database, certificates, and other Docker volumes
docker compose down -v
```

## Configuration

Copy `.env.example` to `.env` and adjust as needed. Common Docker Compose keys:

```bash
KILOCENTER_POSTGRESQL_USERNAME=kilocenter
KILOCENTER_POSTGRESQL_PASSWORD=changeme
KILOCENTER_POSTGRESQL_DATABASE=kilocenter
KILOCENTER_LOG_LEVEL=info
# KILOCENTER_TLS_SERVER_NAME=localhost
```

Set `KILOCENTER_TLS_SERVER_NAME` before the first `docker compose up` if you need a server certificate for a specific hostname instead of `localhost`.

## API Documentation

### gRPC Access

- external gRPC / gRPC-web endpoint: `http://localhost:9090`
- Gateway health: `http://localhost:8087/health`
- KC-Core health: `http://localhost:8086/health`
- KC-Identity health: `http://localhost:8088/health`
- gRPC reflection is enabled by default on KC-Core and KC-Identity for tooling

### Service Surfaces

- `CoreService` - device management, protocol operations, analytics, monitoring, certificates, and SCACI monitoring RPCs
- `IdentityService` - authentication, users, organizations, and API keys
- `KiloCenterService` - compatibility surface that delegates to Core and Identity

In local source development, KC-Web can point its Vite proxy at a different gateway target through `INGRESS_GRPC_URL`. In container deployments, KC-Web is served by nginx and proxies to KC-Gateway automatically.

## Source and Host Deployment

Docker Compose is the recommended install path for evaluation and most deployments.

If you need a non-container workflow, use the dedicated guides instead of ad-hoc manual startup:

- [Linux Host Installation](GitBook/02-GettingStarted/04-installation-linux-host.md)
- [Local VM and Automation](GitBook/02-GettingStarted/05-installation-local-vm-and-automation.md)

### Kubernetes

A Helm chart is provided for Kubernetes deployment. See [helm/kilocenter/](helm/kilocenter/) for the full chart and configuration reference.

```bash
# Install KiloCenter (requires external PostgreSQL and Redis)
helm install kilocenter helm/kilocenter -f my-values.yaml
```

Refer to the [Helm chart README](helm/kilocenter/README.md) for prerequisites, configuration options, and example values.

## MIOTY Protocol Support

KiloCenter implements the MIOTY service-center stack defined by the MIOTY Alliance:

- Physical layer awareness for telegram splitting
- MAC and network-layer message handling
- Unidirectional and bidirectional endpoint support
- Short address handling
- Attachment and detachment flows
- End-to-end encryption support

The community edition focuses on the core service-center stack. Optional MIOTY capabilities such as BSSCI sub-channels, the Variable MAC (VM) attachment, ReCon extensions, and enterprise multi-tenant operation belong to the enterprise offering.

### Supported Profiles and Behaviors

- MIOTY TS-UNB
- Battery-powered endpoint operation
- Multiple redundancy patterns
- Multi-base-station reception handling

## Production Deployment

This repository is not the recommended production rollout vehicle. Use Kilo Cloud or the on-premise Kilo IoT Platform for production deployments that require commercial support, operational guarantees, SLA coverage, multi-tenancy, and optional MIOTY capabilities such as BSSCI sub-channels, Variable MAC (VM), and ReCon extensions.

### System Requirements

- Minimum: 2 CPU cores, 4 GB RAM, 20 GB storage
- Recommended: 4 CPU cores, 8 GB RAM, 100 GB SSD
- OS: Linux (Ubuntu 22.04 LTS recommended)

### Security Considerations

- Always use TLS certificates in production
- Change default credentials before deployment
- Restrict BSSCI (`5000`) and SCACI (`5001`) to trusted networks
- Back up the database regularly
- Monitor system alerts and certificate expiry

## Documentation

- [Getting Started](GitBook/02-GettingStarted/README.md)
- [Architecture and Components](GitBook/02-GettingStarted/01-architecture-and-components.md)
- [Docker Compose Installation](GitBook/02-GettingStarted/03-installation-docker-compose.md)
- [Linux Host Installation](GitBook/02-GettingStarted/04-installation-linux-host.md)
- [Configuration Basics](GitBook/02-GettingStarted/06-configuration-basics.md)
- [API Documentation](GitBook/04-Integrations/02-api.md)
- [MQTT Integration](GitBook/04-Integrations/03-mqtt-first-steps.md)
- [Security](GitBook/05-Security/01-security-and-tenant-isolation-basics.md)

## License

KiloCenter Community Edition is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE).

If you run a modified version as a network service, the AGPL requires you to make the modified source available to users. The source repository is [github.com/Kiloiot/kilo-service-center](https://github.com/Kiloiot/kilo-service-center).

See [TRADEMARKS.md](TRADEMARKS.md) for trademark policy.

Copyright 2024-2026 Tim Kravchunovsky and contributors.

## Support

- **Issues**: [GitHub Issues](https://github.com/Kiloiot/kilo-service-center/issues)
- **Documentation**: [GitBook/](GitBook/)
- **MIOTY Alliance**: [mioty-alliance.com](https://mioty-alliance.com)

## Acknowledgments

KiloCenter is built on standards defined by the MIOTY Alliance and on common operational patterns for self-hosted IoT infrastructure.

---

Refer to the [release notes](https://github.com/Kiloiot/kilo-service-center/releases) for version-specific behavior and upgrade notes.
