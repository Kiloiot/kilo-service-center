# KiloCenter Helm Chart

## Before starting an installation

Use these instructions only in an isolated evaluation environment with test data. The current
examples contain published credentials and signing-key values, and some ports are reachable beyond
the host unless your network blocks them. Changing only the administrator password does not correct
all of these defaults. Do not expose the example installation to the Internet or use it for customer
data. Read the [installation safety notice](../../GitBook/05-Security/02-installation-safety.md) before running commands.


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
# Generate the key-material master key once and store it with your other
# deployment secrets. Every endpoint, session, message and TLS key is
# encrypted at rest under it; the chart refuses to render without it.
openssl rand -hex 32

# Generate the internal peer secret the services present to each other; the
# chart refuses to render without it either.
openssl rand -hex 32

# Add your overrides (at minimum: DB and Redis connection + secrets)
cat > my-values.yaml <<EOF
postgresql:
  host: my-postgres.default.svc.cluster.local
  password: "a-strong-password"

redis:
  host: my-redis.default.svc.cluster.local

secrets:
  masterKey: "<the 64 hex characters from openssl rand -hex 32>"
  internalPeerSecret: "<another 64 hex characters from openssl rand -hex 32>"
  authHmacSecret: "replace-with-a-random-string-at-least-32-bytes"
EOF

# Install
helm install kilocenter ./helm/kilocenter -f my-values.yaml
```

> **Keep the master key.** It stays the same for the lifetime of the
> database: pass the same `secrets.masterKey` to every `helm upgrade`. If it
> is lost, every stored key becomes unreadable and each endpoint and base
> station has to be provisioned again.

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
| `secrets.masterKey` | **Required.** Key-material master key, 64 hex characters or base64 of 32 bytes (`openssl rand -hex 32`). Keep it: stored keys cannot be read without it | `""` |
| `secrets.authHmacSecret` | JWT HMAC signing secret (>= 32 bytes) | dev placeholder |
| `secrets.internalPeerSecret` | **Required**, at least 32 characters (`openssl rand -hex 32`). Shared secret KC-Gateway, KC-Core and KC-Identity present on internal gRPC calls; KC-Core and KC-Identity trust gateway identity headers only from a peer presenting it | `""` |
| `secrets.mqttAdminPassword` | Mosquitto admin password | `KiloCenter` |
| `secrets.mqttClientPassword` | Mosquitto client password | `kilocenter` |

### KC-Core

| Parameter | Description | Default |
|---|---|---|
| `kcCore.image.repository` | Image repository | `ghcr.io/kiloiot/kc-core` |
| `kcCore.image.tag` | Image tag (falls back to `global.imageTag`) | `""` |
| `kcCore.resources` | CPU/memory requests and limits | 10m/128Mi req, 256Mi limit |
| `kcCore.config.logLevel` | Log level | `info` |
| `kcCore.config.bssci.port` | BSSCI listener port | `5000` |
| `kcCore.config.bssci.tlsEnabled` | Enable TLS on BSSCI | `true` |
| `kcCore.config.scaci.port` | SCACI listener port | `5001` |
| `kcCore.config.grpc.port` | Internal gRPC port | `50051` |
| `kcCore.config.health.port` | Health endpoint port | `8086` |
| `kcCore.config.messageRetentionDays` | Message retention in days | `90` |
| `kcCore.config.protocol.connectionEstablishmentTimeout` | BSSCI connection establishment timeout in ms | `30000` |
| `kcCore.config.protocol.socketWriteTimeout` | Bound on every BSSCI and SCACI frame write in ms | `10000` |
| `kcCore.config.protocol.statusRequestInterval` | BSSCI status poll interval in seconds | `30` |
| `kcCore.config.protocol.statusRequestInitialDelay` | Delay before the first status poll in seconds | `5` |
| `kcCore.config.protocol.dlrxQueryTimeout` | DL RX status query timeout in seconds | `300` |
| `kcCore.config.protocol.dlrxCleanupInterval` | DL RX status query cleanup interval in seconds | `60` |
| `kcCore.config.protocol.bsciCertificatePollInterval` | Base station certificate poll interval | `"10s"` |
| `kcCore.config.protocol.scEui` | Service Center EUI; empty defers to env vars or the built-in default | `""` |

### KC-Gateway

| Parameter | Description | Default |
|---|---|---|
| `kcGateway.image.repository` | Image repository | `ghcr.io/kiloiot/kc-gateway` |
| `kcGateway.config.grpc.port` | gRPC-web port | `9090` |
| `kcGateway.config.health.port` | Health endpoint port | `8087` |
| `kcGateway.config.auth.enabled` | Enable authentication | `true` |
| `kcGateway.config.auth.accessTokenTTL` | Access token lifetime | `15m` |
| `kcGateway.config.auth.refreshTokenTTL` | Refresh token lifetime | `24h` |
| `kcGateway.config.corsOrigins` | CORS allowed origins | `[localhost, localhost:5173]` |

### KC-Identity

| Parameter | Description | Default |
|---|---|---|
| `kcIdentity.image.repository` | Image repository | `ghcr.io/kiloiot/kc-identity` |
| `kcIdentity.config.grpc.port` | gRPC port | `50052` |
| `kcIdentity.config.health.port` | Health endpoint port | `8088` |

### KC-Web

| Parameter | Description | Default |
|---|---|---|
| `kcWeb.image.repository` | Image repository | `ghcr.io/kiloiot/kc-web` |
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

### Key Material Conversion (pre-upgrade hook)

On every `helm upgrade` a pre-upgrade hook migrates the schema to `000143`
and runs `rekey -mode=apply`, which converts stored keys to the authenticated
envelope format and reconciles the retired `endpoint_keys` tables. KC-Core and
KC-Identity refuse to migrate past `000143` until that conversion is complete.
The hook reads its credentials from a short-lived Secret that Helm deletes
after the upgrade hooks, because the release Secret still holds the previous
release's values while they run. See the GitBook page *Key material
migration* for the full procedure.

| Parameter | Description | Default |
|---|---|---|
| `rekey.enabled` | Run the pre-upgrade hook | `true` |
| `rekey.mode` | `apply`, `dry-run` or `verify` | `apply` |
| `rekey.migrateToVersion` | Schema version reached before rekeying | `143` |
| `rekey.allowPublishedDevelopmentKey` | Decrypt legacy rows sealed under the published development passphrase. Every install whose deployment never set `KC_ENCRYPTION_KEY` - every install of this chart up to v1.3.0 - holds such rows; they were never secret | `true` |
| `rekey.legacyPassphrase` | The `KC_ENCRYPTION_KEY` passphrase, for a deployment that did set one; requires `rekey.allowPublishedDevelopmentKey=false` | `""` |
| `rekey.archiveExport.existingClaim` | Existing PVC, writable by uid/gid 1000, that receives the encrypted export of `endpoint_keys` rows the hook cannot fold into `endpoints` and of `endpoint_keys_archive` rows. Without it the hook fails when such rows exist | `""` |
| `rekey.archiveExport.mountPath` | Mount path of the export claim; each upgrade writes `endpoint-keys-<revision>.enc` there | `/var/lib/kilocenter/rekey` |

### Rollouts

KC-Core and Mosquitto roll out with the `Recreate` strategy: the old pod stops
before the new one starts. A starting KC-Core marks every BSSCI session its
service center EUI left active as disconnected, so a second KC-Core running
beside the old one would cut off the old pod's live sessions, and both mount a
`ReadWriteOnce` volume (Mosquitto when persistence is enabled) that attaches to
one node at a time. Base
stations reconnect and resume their sessions once the new pod is ready.

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

## Example values — incomplete deployment illustration

```yaml
global:
  imageTag: "v2.0.0"

postgresql:
  host: postgres.database.svc.cluster.local
  password: "strong-db-password"
  sslMode: "require"

redis:
  host: redis.cache.svc.cluster.local

secrets:
  masterKey: "<openssl rand -hex 32, kept for the lifetime of the database>"
  authHmacSecret: "a-cryptographically-random-string-at-least-32-bytes"
  internalPeerSecret: "<openssl rand -hex 32>"
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
