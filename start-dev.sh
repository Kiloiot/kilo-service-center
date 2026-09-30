#!/bin/sh
# Development startup script for KiloCenter (Community Edition)

cd -- "$(dirname -- "$0")" || exit 1

echo "Starting KiloCenter CE development services..."

echo "Stopping any existing processes..."
./dev-services.sh stop

# Export environment variables for PostgreSQL connection
export KILOCENTER_STORAGE_TYPE=postgres
export KILOCENTER_STORAGE_HOST=localhost
export KILOCENTER_STORAGE_DATABASE=kilocenter
export KILOCENTER_STORAGE_USERNAME=kilocenter
export KILOCENTER_STORAGE_PASSWORD=changeme
export KILOCENTER_STORAGE_SSL_MODE=disable

# Key-material master key (mandatory: services fatal without it). The stable
# developer key lives OUTSIDE the repository so it is never committed; it is
# generated once on first run at an XDG path with owner-only permissions.
KILOCENTER_DEV_KEY_FILE="${KILOCENTER_DEV_KEY_FILE:-${XDG_CONFIG_HOME:-$HOME/.config}/kilocenter/master.key}"
if [ ! -f "$KILOCENTER_DEV_KEY_FILE" ]; then
    mkdir -p "$(dirname "$KILOCENTER_DEV_KEY_FILE")"
    ( umask 077; openssl rand -hex 32 > "$KILOCENTER_DEV_KEY_FILE" )
    chmod 600 "$KILOCENTER_DEV_KEY_FILE"
    echo "Generated a development master key at $KILOCENTER_DEV_KEY_FILE"
fi
export KILOCENTER_MASTER_KEY="$(cat "$KILOCENTER_DEV_KEY_FILE")"

# Dev mode settings
export KILOCENTER_SYSTEM_USER_ID="00000000-0000-0000-0000-000000000002"

# Ingress URL — overridable for gateway cutover
export INGRESS_GRPC_URL="${INGRESS_GRPC_URL:-http://localhost:9090}"

# Try to use Docker PostgreSQL if available
echo "Checking Docker PostgreSQL..."
docker ps | grep kilocenter-postgres > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "Using existing Docker PostgreSQL container on port 5433"
    export KILOCENTER_STORAGE_PORT=5433
else
    echo "Docker PostgreSQL not found, using local PostgreSQL on port 5432"
    export KILOCENTER_STORAGE_PORT=5432
    echo "WARNING: PostgreSQL not running. Services will fail to start."
    echo "Please start PostgreSQL manually or use Docker Compose."
fi

# Build KC-Core (CE — no build tags)
echo "Building KC-Core..."
cd KC-Core && go build -o kilocenter ./cmd/kilocenter/ || exit 1
# certgen issues the certificates KC-Core requests (certificates.certgen_path)
go build -o certgen ./cmd/certgen/ || exit 1
cd ..

# Build KC-Identity
echo "Building KC-Identity..."
cd KC-Identity && go build -o identity ./cmd/identity/ || exit 1
cd ..

# Build KC-Gateway
echo "Building KC-Gateway..."
cd KC-Gateway && go build -o gateway ./cmd/gateway/ || exit 1
cd ..

# Start KC-Core (internal mode: loopback :50051, trusts gateway headers)
echo "Starting KC-Core..."
export KILOCENTER_GRPC_PORT=50051
export KILOCENTER_GRPC_HOST=localhost
export KILOCENTER_GRPC_WEB_ENABLED=false
export KILOCENTER_GRPC_INTERNAL_TRUST_ENABLED=true
CORE_PID=$(./dev-services.sh start kc-core)
echo "KC-Core started with PID $CORE_PID (internal :50051)"

# Wait for KC-Core to be ready
echo "Waiting for KC-Core to be ready..."
sleep 3

# Start KC-Identity
echo "Starting KC-Identity..."
export KILOCENTER_GRPC_PORT=50052
export KILOCENTER_GRPC_HOST=localhost
IDENTITY_PID=$(./dev-services.sh start kc-identity)
echo "KC-Identity started with PID $IDENTITY_PID (internal :50052)"

# Wait for KC-Identity to be ready
echo "Waiting for KC-Identity to be ready..."
sleep 3

# Start KC-Gateway (external :9090, authenticates and proxies to KC-Core)
echo "Starting KC-Gateway..."
export KILOCENTER_GRPC_PORT=9090
export KILOCENTER_GRPC_HOST=""
export KILOCENTER_GRPC_WEB_ENABLED=true
export KILOCENTER_GRPC_INTERNAL_TRUST_ENABLED=false
GW_PID=$(./dev-services.sh start kc-gateway)
echo "KC-Gateway started with PID $GW_PID (external :9090)"

# Wait for KC-Gateway to be ready
sleep 2

# Start KC-Web
echo "Starting KC-Web..."
WEB_PID=$(./dev-services.sh start kc-web)
echo "KC-Web started with PID $WEB_PID"

echo ""
echo "Services started:"
echo "  KC-Gateway (gRPC-web): localhost:9090 (PID: $GW_PID)"
echo "  KC-Gateway (Health): http://localhost:8087"
echo "  KC-Core (internal gRPC): localhost:50051 (PID: $CORE_PID)"
echo "  KC-Identity (internal gRPC): localhost:50052 (PID: $IDENTITY_PID)"
echo "  KC-Core (Health): http://localhost:8086"
echo "  KC-Web (Dev): http://localhost:5173 (PID: $WEB_PID)"
echo ""
echo "Logs are in $(dirname "$(./dev-services.sh log kc-core)")"
echo ""
echo "To stop the services, run: ./dev-services.sh stop"
echo "To stop them and the Docker services, run: ./stop-all-services.sh"

# Show initial logs
echo ""
echo "Initial logs:"
echo "============"
sleep 2
for name in kc-gateway kc-core kc-identity kc-web; do
    echo "$name:"
    tail -n 10 "$(./dev-services.sh log "$name")"
    echo ""
done
