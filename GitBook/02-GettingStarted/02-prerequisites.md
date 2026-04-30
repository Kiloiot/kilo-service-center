# Prerequisites

## Required

- Docker and Docker Compose v2
- Git

## Additional Requirements for Building from Source

If you are building KC-Core, KC-Identity, KC-Gateway, or KC-Web from source (development or contribution):

- Go 1.24.4+
- Bun runtime (for KC-Web build and dev server)

## Optional Tools

Useful for validation and troubleshooting:

- `grpcurl` for gRPC API checks
- `mosquitto_sub` and `mosquitto_pub` for MQTT testing (only needed if you enable MQTT integration)
- `psql` for direct database inspection

## Infrastructure Dependencies

KiloCenter requires these services at runtime:

- **PostgreSQL 18+** -- primary data store
- **Redis 7+** -- caching
- **Mosquitto 2.0+** -- MQTT broker (optional for initial setup; MQTT is disabled by default in `KC-Core/config.yaml`)

In the recommended setup, all three run via Docker Compose. Mosquitto is included in the Docker Compose stack but KC-Core will not connect to it unless MQTT is explicitly enabled. See [MQTT First Steps](../04-Integrations/03-mqtt-first-steps.md) for setup instructions.

## Required Ports

Ensure these ports are available before starting:

| Port  | Service             | Notes                         |
|-------|---------------------|-------------------------------|
| 80    | KC-Web (container)  | nginx — SPA + gRPC-web proxy  |
| 9090  | KC-Gateway          | External gRPC / gRPC-web API  |
| 8086  | KC-Core health      | Health and Prometheus          |
| 8087  | KC-Gateway health   | Gateway health endpoint        |
| 8088  | KC-Identity health  | Identity service health        |
| 5000  | BSSCI               | Base station protocol (TLS)    |
| 5001  | SCACI               | Application center protocol    |
| 50051 | KC-Core gRPC        | Internal, not published to host|
| 50052 | KC-Identity gRPC    | Internal, not published to host|
| 5433  | PostgreSQL          | Docker host port mapping       |
| 6379  | Redis               | Cache                          |
| 1883  | MQTT                | Mosquitto broker               |
| 9001  | MQTT WebSocket      | Mosquitto WebSocket            |

Port 5173 is used only in source dev mode (Vite dev server).

## TLS Certificates

KiloCenter requires TLS certificates for base station communication (BSSCI) and
application center communication (SCACI).

- **Docker Compose:** Certificates are generated automatically on first
  `docker compose up`. No manual steps needed. See [Docker Compose Installation](03-installation-docker-compose.md).
- **Linux Host:** Generate certificates manually with `KC-Core/certgen` before
  starting KC-Core. See [Linux Host Installation](04-installation-linux-host.md).

## Repository Layout

Clone the repository:

```bash
git clone https://github.com/Kiloiot/KiloServiceCenter.git
cd KiloServiceCenter
```
