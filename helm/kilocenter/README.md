# KiloCenter Helm Chart

Deploy [KiloCenter](https://github.com/Kiloiot/kilo-service-center) -- an open-source MIOTY network server -- to Kubernetes.

## Prerequisites

| Requirement | Minimum version |
|---|---|
| Kubernetes cluster | 1.25+ |
| Helm | 3.x |
| External PostgreSQL | 14+ |
| External Redis | 7+ |

PostgreSQL and Redis are **not** deployed by this chart. Provision them separately (managed services, operators, or standalone) and supply connection details in your values override.

## Quick Start

```bash
# Add your overrides (at minimum: DB and Redis connection + secrets)
cat > my-values.yaml <<EOF
postgresql:
  host: my-postgres.default.svc.cluster.local
  password: "a-strong-password"

redis:
  host: my-redis.default.svc.cluster.local

secrets:
  authHmacSecret: "replace-with-a-random-string-at-least-32-bytes"
EOF

# Install
helm install kilocenter ./helm/kilocenter -f my-values.yaml
```

## Configuration Reference

### Global

| Parameter | Description | Default |
|---|---|---|
| `global.imageTag` | Default image tag for all components | `latest` |
| `global.imagePullPolicy` | Image pull policy | `IfNotPresent` |
| `global.imagePullSecrets` | Registry pull secrets | `[]` |

### External PostgreSQL

| Parameter | Description | Default |
|---|---|---|
| `postgresql.host` | Hostname | `postgresql` |
| `postgresql.port` | Port | `5432` |
| `postgresql.username` | Username | `kilocenter` |
| `postgresql.password` | Password (override in production) | `changeme` |
| `postgresql.database` | Database name | `kilocenter` |
| `postgresql.sslMode` | SSL mode | `disable` |

### External Redis

| Parameter | Description | Default |
|---|---|---|
| `redis.host` | Hostname | `redis` |
| `redis.port` | Port | `6379` |

### Secrets

| Parameter | Description | Default |
|---|---|---|
| `secrets.authHmacSecret` | JWT HMAC signing secret (>= 32 bytes) | dev placeholder |
| `secrets.mqttAdminPassword` | Mosquitto admin password | `KiloCenter` |
| `secrets.mqttClientPassword` | Mosquitto client password | `kilocenter` |

### KC-Core

| Parameter | Description | Default |
|---|---|---|
| `kcCore.image.repository` | Image repository | `ghcr.io/kiloiot/kiloservicecenter/kc-core` |
| `kcCore.image.tag` | Image tag (falls back to `global.imageTag`) | `""` |
| `kcCore.resources` | CPU/memory requests and limits | 10m/128Mi req, 256Mi limit |
| `kcCore.config.logLevel` | Log level | `info` |
| `kcCore.config.bssci.port` | BSSCI listener port | `5000` |
| `kcCore.config.bssci.tlsEnabled` | Enable TLS on BSSCI | `true` |
| `kcCore.config.scaci.port` | SCACI listener port | `5001` |
| `kcCore.config.grpc.port` | Internal gRPC port | `50051` |
| `kcCore.config.health.port` | Health endpoint port | `8086` |
| `kcCore.config.messageRetentionDays` | Message retention in days | `90` |

### KC-Gateway

| Parameter | Description | Default |
|---|---|---|
| `kcGateway.image.repository` | Image repository | `ghcr.io/kiloiot/kiloservicecenter/kc-gateway` |
| `kcGateway.config.grpc.port` | gRPC-web port | `9090` |
| `kcGateway.config.health.port` | Health endpoint port | `8087` |
| `kcGateway.config.auth.enabled` | Enable authentication | `true` |
| `kcGateway.config.auth.accessTokenTTL` | Access token lifetime | `15m` |
| `kcGateway.config.auth.refreshTokenTTL` | Refresh token lifetime | `24h` |
| `kcGateway.config.corsOrigins` | CORS allowed origins | `[localhost, localhost:5173]` |

### KC-Identity

| Parameter | Description | Default |
|---|---|---|
| `kcIdentity.image.repository` | Image repository | `ghcr.io/kiloiot/kiloservicecenter/kc-identity` |
| `kcIdentity.config.grpc.port` | gRPC port | `50052` |
| `kcIdentity.config.health.port` | Health endpoint port | `8088` |

### KC-Web

| Parameter | Description | Default |
|---|---|---|
| `kcWeb.image.repository` | Image repository | `ghcr.io/kiloiot/kiloservicecenter/kc-web` |
| `kcWeb.resources` | CPU/memory requests and limits | 5m/32Mi req, 64Mi limit |

### Mosquitto

| Parameter | Description | Default |
|---|---|---|
| `mosquitto.image.tag` | Eclipse Mosquitto image tag | `2.0.22` |
| `mosquitto.persistence.enabled` | Enable persistent storage | `true` |
| `mosquitto.persistence.size` | PVC size | `1Gi` |

### TLS Certificate Generation

| Parameter | Description | Default |
|---|---|---|
| `certgen.serverName` | Server name for generated certs | `localhost` |
| `certPvc.size` | Certificate PVC size | `100Mi` |

### Ingress

| Parameter | Description | Default |
|---|---|---|
| `ingress.enabled` | Enable Kubernetes Ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Host rules | see values.yaml |
| `ingress.tls` | TLS configuration | `[]` |

## TLS Certificates

On first install, a Helm pre-install hook Job runs the `certgen` binary to generate a self-signed CA and server certificate into a shared PVC. All subsequent installs and upgrades skip generation if certificates already exist.

For production deployments, replace the generated certificates with ones signed by a trusted CA by mounting your own secret or PVC at `/app/certificates` in the kc-core pod.

## Example: Minimal Production Override

```yaml
global:
  imageTag: "1.0.0"

postgresql:
  host: postgres.database.svc.cluster.local
  password: "strong-db-password"
  sslMode: "require"

redis:
  host: redis.cache.svc.cluster.local

secrets:
  authHmacSecret: "a-cryptographically-random-string-at-least-32-bytes"
  mqttAdminPassword: "strong-mqtt-admin-pw"
  mqttClientPassword: "strong-mqtt-client-pw"

certgen:
  serverName: "kilocenter.example.com"

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: kilocenter.example.com
      paths:
        - path: /
          pathType: Prefix
          service: kc-web
          port: 80
        - path: /kilocenter.api
          pathType: Prefix
          service: kc-gateway
          port: 9090
  tls:
    - secretName: kilocenter-tls
      hosts:
        - kilocenter.example.com
```

## Architecture

```
                  Internet
                     |
               [ Ingress ] (optional)
                /         \
         kc-web:80    kc-gateway:9090
                          |
                    kc-core:50051 ---- kc-identity:50052
                    /      |      \
             bssci:5000  scaci:5001  mosquitto:1883
                                         |
                                    [MQTT clients]
```

- **kc-core** -- BSSCI/SCACI protocol engines, internal gRPC server
- **kc-gateway** -- external gRPC-web ingress, authentication proxy
- **kc-identity** -- user/org authentication and management
- **kc-web** -- static browser UI served by nginx
- **mosquitto** -- MQTT broker for IoT message dispatch
