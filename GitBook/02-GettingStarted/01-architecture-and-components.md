# Architecture and Components

## Runtime Services

KiloCenter runs as four cooperating services plus infrastructure dependencies:

- **KC-Core** -- MIOTY service center engine. Handles BSSCI (base station protocol) and SCACI (application center protocol), processes uplink/downlink data, and exposes an internal gRPC API.
- **KC-Identity** -- Identity and organization service. Provides user authentication, organization management, API key management, and tenant isolation.
- **KC-Gateway** -- External API ingress. Provides gRPC and gRPC-web access for KC-Web and external clients. Proxies requests to KC-Core and KC-Identity.
- **KC-Web** -- Browser-based operator interface for managing base stations, endpoints, and monitoring traffic. Served by nginx in container mode.

## Library Modules

These modules are compiled into the runtime services above and are not deployed as separate containers:

- **KC-DB** -- Persistence module used by KC-Core and KC-Identity. Contains data models, repository interfaces, and PostgreSQL migrations.
- **KC-MQTT** -- MQTT integration module used by KC-Core. Publishes uplink data and device events to MQTT topics.

## Infrastructure Dependencies

All editions require:

- **PostgreSQL** -- primary data store for messages, endpoint state, and configuration
- **Redis** -- caching and session storage
- **Mosquitto** -- MQTT broker for real-time data streaming to external consumers

## Component Tree

```text
KiloCenter/
├── KC-Core/            - MIOTY service center engine
├── KC-Identity/        - Identity and organization service
├── KC-Gateway/         - External gRPC-web API gateway
├── KC-Web/             - Browser operator interface
├── KC-DB/              - Data models, repositories, migrations
├── KC-MQTT/            - MQTT topic and dispatch integration
├── config/             - Runtime configuration files
├── docker-compose.yml  - Container runtime definitions
└── deployments/docker/ - Mosquitto, nginx, and Docker support assets
```

## Data Path

1. MIOTY endpoints transmit data over the air.
2. Base stations receive radio frames and connect to KC-Core over BSSCI (TCP with TLS on port 5000).
3. KC-Core validates protocol frames, manages sessions, and persists messages through KC-DB.
4. KC-Gateway exposes the gRPC-web API on port 9090, proxying requests to KC-Core.
5. KC-Web connects to KC-Gateway for all operator workflows (container mode: nginx on port 80 proxies gRPC-web to KC-Gateway).
6. External systems consume data through gRPC (via KC-Gateway) or MQTT (via Mosquitto).

## Service Ports

| Service              | Port  | Protocol   | Description                     |
|----------------------|-------|------------|---------------------------------|
| KC-Core (internal)   | 50051 | gRPC       | Internal API (KC-Gateway only)  |
| KC-Core (health)     | 8086  | HTTP       | Health and Prometheus metrics    |
| KC-Core (BSSCI)      | 5000  | TCP/TLS    | Base station protocol ingress    |
| KC-Core (SCACI)      | 5001  | TCP/TLS    | Application center protocol      |
| KC-Identity (gRPC)   | 50052 | gRPC       | Internal only (KC-Core/Gateway)  |
| KC-Identity (health) | 8088  | HTTP       | Identity service health          |
| KC-Gateway           | 9090  | gRPC-web   | External API ingress             |
| KC-Gateway (health)  | 8087  | HTTP       | Gateway health endpoint          |
| KC-Web (container)   | 80    | HTTP       | nginx — serves SPA, proxies gRPC |
| KC-Web (source dev)  | 5173  | HTTP       | Vite dev server                  |
| PostgreSQL           | 5433  | TCP        | Database (Docker host mapping)   |
| Redis                | 6379  | TCP        | Cache                            |
| Mosquitto            | 1883  | TCP        | MQTT broker                      |

## Local Deployment Modes

- **Container mode** (recommended): `docker compose up --build -d` runs all services in containers. KC-Web is served by nginx at `http://localhost/`. No host toolchain required beyond Docker.
- **Source dev mode**: Docker for infrastructure dependencies (postgres, redis, mosquitto); KC-Core, KC-Identity, KC-Gateway, and KC-Web built and run from source. KC-Web available at `http://localhost:5173`.
