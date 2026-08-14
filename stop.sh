#!/usr/bin/env bash
#
# Stop the backend and UI.
#   --nodes : also stop the 3 mpcium node processes
#   --all   : also stop nodes AND the local Docker NATS/Consul
#
set -uo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "Stopping backend + UI…"
pkill -f "stellar-wallet-backend" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
rm -f "$ROOT/logs/backend.pid" 2>/dev/null || true

if [ "${1:-}" = "--nodes" ] || [ "${1:-}" = "--all" ]; then
  echo "Stopping mpcium nodes…"
  pkill -f "mpcium start" 2>/dev/null || true
fi

if [ "${1:-}" = "--all" ]; then
  echo "Stopping local Docker NATS/Consul…"
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$ROOT/infra/docker-compose.yaml" down 2>/dev/null || true
  else
    docker-compose -f "$ROOT/infra/docker-compose.yaml" down 2>/dev/null || true
  fi
fi

echo "Done."
