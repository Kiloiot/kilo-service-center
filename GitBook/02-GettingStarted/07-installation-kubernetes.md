# Installation: Kubernetes (Helm)

## Goal

Deploy KiloCenter to a Kubernetes cluster using the Helm chart included in this repository.

## Prerequisites

| Requirement | Minimum version |
|---|---|
| Kubernetes cluster | 1.25+ |
| Helm | 3.x |
| External PostgreSQL | 14+ |
| External Redis | 7+ |

PostgreSQL and Redis are **not** deployed by this chart. Provision them separately (managed services, operators, or standalone) and supply connection details in your values override.

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

The chart deploys five services and one MQTT broker:

| Component | Port(s) | Description |
|-----------|---------|-------------|
| `kc-core` | 50051 (gRPC), 5000 (BSSCI), 5001 (SCACI), 8086 (health) | Service center engine |
| `kc-gateway` | 9090 (gRPC-web), 8087 (health) | External API ingress |
| `kc-identity` | 50052 (gRPC), 8088 (health) | Identity, users, organizations |
| `kc-web` | 80 | Web management interface (nginx) |
| `mosquitto` | 1883, 9001 (WebSocket) | MQTT broker |
| `certgen` | — | Pre-install hook that generates TLS certificates |

## Step 1: Create a Values Override

At minimum, override the database, Redis, and secret settings:

```yaml
# my-values.yaml
postgresql:
  host: my-postgres.default.svc.cluster.local
  password: "a-strong-password"
  sslMode: "require"

redis:
  host: my-redis.default.svc.cluster.local

secrets:
  authHmacSecret: "replace-with-a-random-string-at-least-32-bytes"
  mqttAdminPassword: "strong-mqtt-admin-pw"
  mqttClientPassword: "strong-mqtt-client-pw"

certgen:
  serverName: "kilocenter.example.com"
```

> **Important:** The `authHmacSecret` is used to sign and verify JWT tokens across KC-Gateway and KC-Identity. It must be at least 32 characters.

## Step 2: Install

```bash
helm install kilocenter ./helm/kilocenter -f my-values.yaml
```

On first install, a pre-install hook Job runs the `certgen` binary to generate a self-signed CA and server certificate into a shared PVC. Subsequent upgrades skip generation if certificates already exist.

## Step 3: Validate

```bash
# Check all pods are running
kubectl get pods -l app.kubernetes.io/instance=kilocenter

# Check service health
kubectl exec deploy/kilocenter-kc-core -- wget -qO- http://localhost:8086/health/ping
kubectl exec deploy/kilocenter-kc-identity -- wget -qO- http://localhost:8088/health
kubectl exec deploy/kilocenter-kc-gateway -- wget -qO- http://localhost:8087/health
```

## Step 4: Access the UI

Without ingress, use port-forwarding:

```bash
kubectl port-forward svc/kilocenter-kc-web 8080:80
```

Then open [http://localhost:8080/](http://localhost:8080/) in your browser.

## Default Admin Account

On first startup, a default admin user is created via database migration:

| | |
|---|---|
| **Email** | `admin@kilocenter.local` |
| **Password** | `admin123!` |

> **Warning:** Change the password or remove this account before any public-facing deployment. The credentials are published in this repository.

## Ingress

Enable standard Kubernetes Ingress in your values override:

```yaml
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

When using ingress, add your domain to the CORS allowed origins:

```yaml
kcGateway:
  config:
    corsOrigins:
      - "https://kilocenter.example.com"
```

## BSSCI/SCACI Protocol Access

Base stations connect directly to KC-Core over TCP+TLS on ports 5000 (BSSCI) and 5001 (SCACI). These are raw TCP connections, not HTTP. To expose them externally, create a LoadBalancer service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kilocenter-bssci
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: kc-core
    app.kubernetes.io/instance: kilocenter
  ports:
    - name: bssci
      port: 5000
      targetPort: 5000
    - name: scaci
      port: 5001
      targetPort: 5001
```

## TLS Certificates

The `certgen` hook generates a self-signed CA and server certificate on first install. For production, replace these with certificates signed by a trusted CA by mounting your own secret or PVC at `/app/certificates` in the kc-core pod.

## Configuration Reference

For the full list of configurable parameters, see the [Helm chart README](../../helm/kilocenter/README.md).

## Upgrading

```bash
helm upgrade kilocenter ./helm/kilocenter -f my-values.yaml
```

Set a specific image tag to pin a version:

```yaml
global:
  imageTag: "1.0.0"
```

## Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Pods in `ImagePullBackOff` | Missing image pull secret | Add `global.imagePullSecrets` with your registry credentials |
| KC-Core readiness probe 503 | Dependency not ready | Check KC-Identity and PostgreSQL are running |
| `invalid_token` after login | HMAC secret mismatch | Ensure `secrets.authHmacSecret` is set (same for gateway and identity) |
| BSSCI connection refused | No external service | Create a LoadBalancer service for ports 5000/5001 |
| gRPC-web errors in browser | CORS or ingress misconfigured | Check `kcGateway.config.corsOrigins` and ingress paths |
