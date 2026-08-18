#!/usr/bin/env bash
#
# Stop the Docker Compose stack.
#   ./stop.sh          # stop + remove containers (volumes kept: wallet DB + key-shares)
#   ./stop.sh --all    # also remove volumes (wipes wallet DB + key-shares)
#
set -uo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
else
  COMPOSE=(docker-compose)
fi

if [ "${1:-}" = "--all" ]; then
  echo "Stopping stack + removing volumes…"
  "${COMPOSE[@]}" down -v --remove-orphans
else
  echo "Stopping stack (volumes kept)…"
  "${COMPOSE[@]}" down --remove-orphans
fi

echo "Done."
