#!/bin/bash
# Development startup script for KiloCenter (Community Edition)

echo "Starting KiloCenter CE development services..."

# Kill any existing processes
echo "Stopping any existing processes..."
pkill -f "KC-Core/kilocenter" || true
pkill -f "KC-Gateway/gateway" || true
pkill -f "bun.*dev.*KC-Web" || true

# Wait for processes to stop
sleep 2

# Create logs directories
mkdir -p ../logs/runtime ../logs/pids

# Export environment variables for PostgreSQL connection
export KILOCENTER_STORAGE_TYPE=postgres
export KILOCENTER_STORAGE_HOST=localhost
export KILOCENTER_STORAGE_DATABASE=kilocenter
export KILOCENTER_STORAGE_USERNAME=kilocenter
export KILOCENTER_STORAGE_PASSWORD=changeme
export KILOCENTER_STORAGE_SSL_MODE=disable

# Dev mode settings
export KILOCENTER_ALLOW_PLAINTEXT_KEYS=true
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
cd KC-Core
./kilocenter -config config.yaml > ../../logs/runtime/kc-core.log 2>&1 &
CORE_PID=$!
echo "$CORE_PID" > ../../logs/pids/kc-core.pid
echo "KC-Core started with PID $CORE_PID (internal :50051)"
cd ..

# Wait for KC-Core to be ready
echo "Waiting for KC-Core to be ready..."
sleep 3

# Start KC-Identity
echo "Starting KC-Identity..."
export KILOCENTER_GRPC_PORT=50052
export KILOCENTER_GRPC_HOST=localhost
cd KC-Identity
./identity -config config.yaml > ../../logs/runtime/kc-identity.log 2>&1 &
IDENTITY_PID=$!
echo "$IDENTITY_PID" > ../../logs/pids/kc-identity.pid
echo "KC-Identity started with PID $IDENTITY_PID (internal :50052)"
cd ..

# Wait for KC-Identity to be ready
echo "Waiting for KC-Identity to be ready..."
sleep 3

# Start KC-Gateway (external :9090, authenticates and proxies to KC-Core)
echo "Starting KC-Gateway..."
export KILOCENTER_GRPC_PORT=9090
export KILOCENTER_GRPC_HOST=""
export KILOCENTER_GRPC_WEB_ENABLED=true
export KILOCENTER_GRPC_INTERNAL_TRUST_ENABLED=false
cd KC-Gateway
./gateway -config config.yaml > ../../logs/runtime/kc-gateway.log 2>&1 &
GW_PID=$!
echo "$GW_PID" > ../../logs/pids/kc-gateway.pid
echo "KC-Gateway started with PID $GW_PID (external :9090)"
cd ..

# Wait for KC-Gateway to be ready
sleep 2

# Start KC-Web
echo "Starting KC-Web..."
cd KC-Web
bun run dev > ../../logs/runtime/kc-web.log 2>&1 &
WEB_PID=$!
echo "$WEB_PID" > ../../logs/pids/kc-web.pid
echo "KC-Web started with PID $WEB_PID"
cd ..

echo ""
echo "Services started:"
echo "  KC-Gateway (gRPC-web): localhost:9090 (PID: $GW_PID)"
echo "  KC-Gateway (Health): http://localhost:8087"
echo "  KC-Core (internal gRPC): localhost:50051 (PID: $CORE_PID)"
echo "  KC-Core (Health): http://localhost:8086"
echo "  KC-Web (Dev): http://localhost:5173 (PID: $WEB_PID)"
echo ""
echo "Logs are in logs/runtime/"
echo ""
echo "To stop all services, run: ./stop-all-services.sh"

# Show initial logs
echo ""
echo "Initial logs:"
echo "============"
sleep 2
echo "KC-Gateway:"
tail -n 10 ../../logs/runtime/kc-gateway.log
echo ""
echo "KC-Core:"
tail -n 10 ../../logs/runtime/kc-core.log
echo ""
echo "KC-Web:"
tail -n 10 ../../logs/runtime/kc-web.log
