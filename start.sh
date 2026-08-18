#!/usr/bin/env bash
#
# One-command full stack on Docker Compose: generates the mpcium keys if they're
# missing, then builds & starts NATS + Consul + 3 mpcium nodes + backend + UI.
# When it's done, open http://localhost:8080.
#
#   ./start.sh           # up (generate keys if missing)
#   ./start.sh --fresh   # wipe volumes + regenerate keys, clean start
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

say()  { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
ok()   { printf "\033[1;32m✓ %s\033[0m\n" "$1"; }
warn() { printf "\033[1;33m! %s\033[0m\n" "$1"; }
die()  { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

# Pick the compose command (plugin form or standalone).
if docker compose version >/dev/null 2>&1; then
  COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE=(docker-compose)
else
  die "docker compose not found (install Docker Compose)"
fi

# Docker daemon must be reachable.
docker info >/dev/null 2>&1 || die "Docker daemon not reachable — start it first (e.g. 'colima start')"

FRESH=0
[ "${1:-}" = "--fresh" ] && FRESH=1

if [ "$FRESH" = "1" ]; then
  say "Fresh start — tearing down stack + volumes"
  "${COMPOSE[@]}" down -v --remove-orphans 2>/dev/null || true
  say "Regenerating keys"
  FORCE=1 ./scripts/gen-keys.sh
else
  # Generate keys only if they don't exist yet (idempotent).
  ./scripts/gen-keys.sh
fi

say "Building & starting the stack"
"${COMPOSE[@]}" up -d --build

# Wait for the 3 nodes to report healthy; if Docker DNS was flaky on a fresh
# VM the nodes may crash-loop once — restart them and re-check.
healthy() {
  local up=0
  for p in 8091 8092 8093; do
    curl -s -m 2 "http://localhost:$p/health" | grep -q '"live":true' && up=$((up + 1))
  done
  [ "$up" -eq 3 ]
}

say "Waiting for mpcium nodes to become healthy"
tries=0
until healthy; do
  tries=$((tries + 1))
  if [ "$tries" = "12" ]; then
    warn "nodes not healthy yet — restarting them (Docker DNS can be flaky on first boot)"
    "${COMPOSE[@]}" restart node0 node1 node2 >/dev/null 2>&1 || true
  fi
  [ "$tries" -gt 30 ] && { warn "nodes still not healthy — check: ${COMPOSE[*]} logs node0"; break; }
  sleep 2
done

if healthy; then
  ok "All 3 mpcium nodes healthy"
fi

ok "Stack is up → open http://localhost:8080"
echo "   backend :8090   consul :8500   node health :8091-8093"
echo "   logs:  ${COMPOSE[*]} logs -f backend"
echo "   stop:  ./stop.sh        (add --all to also remove volumes)"
