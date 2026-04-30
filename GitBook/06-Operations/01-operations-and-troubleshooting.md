# Operations and Troubleshooting

## Goal

Run repeatable checks for startup issues, connectivity problems, and message-flow failures.

## Health Checks

From `kilocenter-modules/`:

```bash
# KC-Core health
curl -s http://localhost:8086/health

# KC-Gateway health
curl -s http://localhost:8087/health

# gRPC service health
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check

# All container services
docker compose ps
```

## Port Checks

Verify all expected services are listening:

```bash
lsof -i :80     # KC-Web (container mode)
lsof -i :9090   # KC-Gateway (gRPC-web)
lsof -i :8086   # KC-Core health
lsof -i :8087   # KC-Gateway health
lsof -i :5000   # BSSCI
lsof -i :5001   # SCACI
lsof -i :50051  # KC-Core internal gRPC
lsof -i :5433   # PostgreSQL (Docker)
lsof -i :6379   # Redis
lsof -i :1883   # MQTT (if enabled)
```

Port 5173 is used only in source dev mode (Vite dev server).

## Container Mode Operations

```bash
# View logs for a service
docker compose logs -f kilocenter
docker compose logs -f kc-gateway
docker compose logs -f kc-web

# Restart a single service
docker compose restart kilocenter

# Check all service states
docker compose ps
```

## Log Files (Source Dev Mode)

All service logs are written to `kilocenter-modules/logs/runtime/`:

```bash
# KC-Core logs (BSSCI activity, message processing)
tail -f logs/runtime/kc-core.log

# KC-Gateway logs (API proxy activity)
tail -f logs/runtime/kc-gateway.log

# KC-Web logs (frontend dev server)
tail -f logs/runtime/kc-web.log

# Search for errors across all logs
grep -E "ERROR|FATAL" logs/runtime/*.log
```

## Database Connection Check

Verify PostgreSQL is accessible:

```bash
PGPASSWORD=changeme psql -U kilocenter -h localhost -p 5433 -d kilocenter -c "SELECT 1;"
```

Use port `5432` instead of `5433` if running PostgreSQL directly on the host (not via Docker).

## Stopping Services

**Container mode:**

```bash
docker compose down
```

**Source dev mode:**

```bash
./stop-all-services.sh

# Also stop infrastructure containers
docker compose stop postgres redis mosquitto
```

## Common Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| `dial tcp ...5432: connection refused` | PostgreSQL port mismatch | Use `5433` for Docker, `5432` for host install |
| KC-Web shows gRPC errors | KC-Gateway not running or wrong port | Verify KC-Gateway on port 9090; check nginx config in container mode |
| `permission denied /var/run/docker.sock` | Docker group membership | Run `sudo usermod -aG docker $USER` and re-login |
| `failed to load TLS certificate` | Missing or invalid certificate files | Run `certgen` -- see [Installation](../02-GettingStarted/03-installation-docker-compose.md) |
| Base station not connecting | BSSCI port 5000 not listening | Check KC-Core started and TLS certs are configured |
| SCACI not accepting connections | Missing `scaci_tls` config block | Check `config/config.docker.yaml` has `scaci_host` and `scaci_tls` section |
| KC-Web returns 502 for gRPC calls | nginx cannot reach `kc-gateway:9090` | Verify `kc-gateway` is healthy: `docker compose ps` |
| `start-dev.sh` fails | Dependencies not running | Start `docker compose up -d postgres redis mosquitto` first |
| Bun not found | Bun runtime not installed | Install Bun: `curl -fsSL https://bun.sh/install \| bash` |
| KC-Web loads but shows no data | KC-Core not running or no base stations connected | Check health endpoints and base station status |

## Recovery Sequence

**Container mode:**

1. Stop all services: `docker compose down`
2. Start all services: `docker compose up -d`
3. Run health checks (see above)

**Source dev mode:**

1. Stop all services: `./stop-all-services.sh`
2. Restart infrastructure: `docker compose restart postgres redis mosquitto`
3. Start services: `./start-dev.sh`
4. Run health checks (see above)
