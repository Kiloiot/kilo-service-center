#!/bin/bash

# KiloCenter MIOTY System - Stop All Services Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Stopping KiloCenter MIOTY System..."

# Function to stop a service by PID file
stop_service() {
    local name=$1
    local pidfile=$2

    if [ -f "$pidfile" ]; then
        pid=$(cat "$pidfile")
        if kill -0 "$pid" 2>/dev/null; then
            echo "Stopping $name (PID: $pid)..."
            kill "$pid"
            rm "$pidfile"
            echo "✓ $name stopped"
        else
            echo "✗ $name was not running (stale PID file)"
            rm "$pidfile"
        fi
    else
        echo "✗ $name PID file not found"
    fi
}

# Stop services in reverse order
echo ""
echo "=== Stopping Services ==="

# Stop KC-Web
stop_service "KC-Web" "../logs/pids/kc-web.pid"

# Stop KC-Gateway
stop_service "KC-Gateway" "../logs/pids/kc-gateway.pid"

# Stop KC-Identity
stop_service "KC-Identity" "../logs/pids/kc-identity.pid"

# Stop KC-Core
stop_service "KC-Core" "../logs/pids/kc-core.pid"

# Also kill any remaining Go processes (in case PIDs weren't tracked)
echo ""
echo "=== Cleaning up any remaining processes ==="
pkill -f "go run.*kilocenter" || true
pkill -f "KC-Identity/identity" || true
pkill -f "KC-Gateway/gateway" || true
pkill -f "bun.*dev" || true

# Stop Docker services
echo ""
echo "=== Stopping Docker Services ==="
docker compose down

echo ""
echo "✓ All services stopped"
