#!/bin/sh

# KiloCenter MIOTY System - Stop All Services Script

cd -- "$(dirname -- "$0")" || exit 1

echo "Stopping KiloCenter MIOTY System..."

echo ""
echo "=== Stopping Services ==="
./dev-services.sh stop

# Stop Docker services
echo ""
echo "=== Stopping Docker Services ==="
docker compose down

echo ""
echo "All services stopped"
