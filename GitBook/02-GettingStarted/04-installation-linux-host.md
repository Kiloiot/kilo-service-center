# Installation: Linux Host

## Before starting an installation

Use these instructions only in an isolated evaluation environment with test data. The current
examples contain published credentials and signing-key values, and some ports are reachable beyond
the host unless your network blocks them. Changing only the administrator password does not correct
all of these defaults. Do not expose the example installation to the Internet or use it for customer
data. Read the [installation safety notice](../05-Security/02-installation-safety.md) before running commands.


## Goal

Run KiloCenter without Docker by installing all dependencies directly on the host.

## Step 1: Install Infrastructure Dependencies

Install and start these services on the host:

- **PostgreSQL 18+** -- create a database named `kilocenter`
- **Redis 7+**
- **Mosquitto 2.0+**

## Step 2: Generate TLS Certificates

Build the certificate generator and generate certificates before configuring or starting KC-Core:

```bash
cd kilo-service-center
go build -o KC-Core/certgen KC-Core/cmd/certgen/main.go
KC-Core/certgen -dir KC-Core/certificates -days 365 -server your-hostname.example.com
```

Replace `your-hostname.example.com` with the FQDN or IP address that base stations will use to reach this server. For local-only testing, use `localhost`.

This creates four files in `KC-Core/certificates/`:

- `ca.crt` and `ca.key` -- CA certificate and private key
- `server.crt` and `server.key` -- server certificate and private key

The server certificate automatically includes `localhost`, `127.0.0.1`, and all local network IPs as SANs.

KC-Core will fail to start without these files. To renew later without regenerating the CA, use `-server-only`:

```bash
KC-Core/certgen -dir KC-Core/certificates -days 365 -server your-hostname.example.com -server-only
```

See [Security](../05-Security/01-security-and-tenant-isolation-basics.md) for full certgen reference and client certificate generation.

## Step 3: Configure KC-Core

Update `KC-Core/config.yaml` to match your host service addresses. Key settings:

- `storage.host` and `storage.port` -- PostgreSQL address (typically `localhost:5432` for host install)
- `storage.username` and `storage.password` -- database credentials
- BSSCI and SCACI TLS certificate paths

## Step 4: Create the Master Key

KC-Core and KC-Identity encrypt every endpoint, session, message and TLS key
at rest under a master key read from the `KILOCENTER_MASTER_KEY` environment
variable, and refuse to start without it. Generate it once and keep it in a
file only your user can read:

```bash
mkdir -p ~/.config/kilocenter
(umask 077 && openssl rand -hex 32 > ~/.config/kilocenter/master.key)
```

Export it in every shell (or service unit) that starts KC-Core or KC-Identity;
both must receive the same value:

```bash
export KILOCENTER_MASTER_KEY="$(cat ~/.config/kilocenter/master.key)"
```

Back the file up with your other secrets and never replace it for an existing
database: if it is lost, the stored keys cannot be decrypted and every
endpoint and base station has to be provisioned again.

## Step 5: Build and Start KC-Core

From `KC-Core/`, with `KILOCENTER_MASTER_KEY` exported:

```bash
go build -o kilocenter ./cmd/kilocenter/
./kilocenter -config config.yaml
```

## Step 6: Build and Start KC-Identity

From `KC-Identity/`, with `KILOCENTER_MASTER_KEY` exported:

```bash
go build -o identity ./cmd/identity/
./identity -config config.yaml
```

KC-Identity provides user authentication, organization management, and API key services on port 50052.

## Step 7: Build and Start KC-Gateway

From `KC-Gateway/`:

```bash
go build -o gateway ./cmd/gateway/
./gateway -config config.yaml
```

KC-Gateway proxies external gRPC-web requests to KC-Core and KC-Identity. It must be started after both upstream services are healthy.

The shipped configurations keep KC-Core and KC-Identity on `localhost`, so the
internal calls between the three services never leave the host. If you bind
either service to another address (`grpc.host`) or run them on separate hosts,
generate a shared secret with `openssl rand -hex 32` and export it as
`KILOCENTER_INTERNAL_AUTH_PEER_SECRET` for all three: the services refuse to
start without one of at least 32 characters.

## Step 8: Start KC-Web

From `KC-Web/`:

```bash
bun install
bun run dev
```

## Step 9: Validate

- KC-Core health: [http://localhost:8086/health](http://localhost:8086/health)
- KC-Identity health: [http://localhost:8088/health](http://localhost:8088/health)
- KC-Gateway health: [http://localhost:8087/health](http://localhost:8087/health)
- KC-Web: [http://localhost:5173](http://localhost:5173)
