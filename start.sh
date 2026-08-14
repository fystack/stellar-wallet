#!/usr/bin/env bash
#
# Seamless start: MPC cluster (NATS+Consul+3 nodes) + Go backend + Vite UI.
#
# Infra is automatic:
#   • existing cluster + shared NATS/Consul (10.10.0.1) reachable → use it
#   • otherwise → boot a LOCAL NATS + Consul via Docker
#   • no mpcium/ folder at all → bootstrap a fresh cluster from scratch
#     (installs mpcium/mpcium-cli, generates peers/identities/initiator)
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
MPCIUM="$ROOT/mpcium"
LOGS="$ROOT/logs"
COMPOSE="$ROOT/infra/docker-compose.yaml"
mkdir -p "$LOGS"

SHARED_HOST="${SHARED_HOST:-10.10.0.1}"
NATS_PORT="${NATS_PORT:-4222}"
CONSUL_PORT="${CONSUL_PORT:-8500}"
BACKEND_ADDR="${BACKEND_ADDR:-:8090}"
UI_PORT="${UI_PORT:-5173}"
export NATS_PORT CONSUL_PORT

say() { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
ok()  { printf "\033[1;32m✓ %s\033[0m\n" "$1"; }
die() { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

command -v go  >/dev/null || die "go not found"
command -v npm >/dev/null || die "npm not found"

reachable() { nc -z -w2 "$1" "$2" >/dev/null 2>&1; }

# Support both the compose plugin and the standalone binary.
compose() {
  if docker compose version >/dev/null 2>&1; then docker compose "$@"
  else docker-compose "$@"; fi
}

boot_local_infra() {
  command -v docker >/dev/null || die "docker not found (needed for local infra)"
  command -v docker-compose >/dev/null || docker compose version >/dev/null 2>&1 || die "docker compose not found"
  say "Booting local NATS + Consul (Docker)"
  compose -f "$COMPOSE" up -d >/dev/null 2>&1 || die "docker compose up failed"
  for i in $(seq 1 40); do
    if reachable 127.0.0.1 "$NATS_PORT" &&
       [ -n "$(curl -s "http://127.0.0.1:$CONSUL_PORT/v1/status/leader" | tr -d '"')" ]; then
      return 0
    fi
    sleep 1
  done
  die "local NATS/Consul did not become ready (ports $NATS_PORT/$CONSUL_PORT). If busy, set NATS_PORT/CONSUL_PORT."
}

sedi() { if sed --version >/dev/null 2>&1; then sed -i "$@"; else sed -i '' "$@"; fi; }
align_host() {
  local f="$1"
  [ -f "$f" ] || return 0
  sedi -E "s|url: nats://[^:]+:4222|url: nats://$HOST:4222|" "$f"
  sedi -E "s|address: [0-9a-zA-Z.]+:8500|address: $HOST:8500|" "$f"
}

# --- 1) Infra + cluster setup -----------------------------------------------
if [ ! -f "$MPCIUM/node0/config.yaml" ]; then
  # Fresh checkout — bootstrap a self-contained local cluster.
  say "No cluster found — bootstrapping a local one"
  boot_local_infra
  HOST="127.0.0.1"
  HOST="$HOST" bash "$ROOT/bootstrap.sh" || die "bootstrap failed"
elif reachable "$SHARED_HOST" "$NATS_PORT" && reachable "$SHARED_HOST" "$CONSUL_PORT"; then
  HOST="$SHARED_HOST"
  ok "Using shared cluster at $HOST"
else
  boot_local_infra
  HOST="127.0.0.1"
  align_host "$MPCIUM/config.yaml"
  for n in 0 1 2; do align_host "$MPCIUM/node$n/config.yaml"; done
  if ! curl -s "http://$HOST:$CONSUL_PORT/v1/kv/mpc_peers/?keys" | grep -q node0; then
    say "Registering peers into Consul"
    ( cd "$MPCIUM" && mpcium-cli register-peers ) || die "register-peers failed"
  fi
  ok "Using local cluster at $HOST"
fi

# --- 2) Start the 3 MPC nodes (only if not already running) -----------------
for n in 0 1 2; do
  if pgrep -f "mpcium start -n node$n" >/dev/null; then
    say "node$n already running"
  else
    say "Starting node$n"
    ( cd "$MPCIUM/node$n" && mpcium start -n "node$n" > "$LOGS/node$n.log" 2>&1 & )
  fi
done
sleep 4

# --- 3) Backend -------------------------------------------------------------
say "Building & starting backend on $BACKEND_ADDR"
( cd "$ROOT/backend" && go build -o stellar-wallet-backend . ) || die "backend build failed"
ADDR="$BACKEND_ADDR" NATS_URL="nats://$HOST:$NATS_PORT" CONSUL_ADDR="$HOST:$CONSUL_PORT" \
  INITIATOR_KEY="$MPCIUM/event_initiator.key" DB_PATH="$ROOT/backend/wallet.db" \
  GIN_MODE=release \
  "$ROOT/backend/stellar-wallet-backend" > "$LOGS/backend.log" 2>&1 &
BACKEND_PID=$!
echo "$BACKEND_PID" > "$LOGS/backend.pid"
sleep 2
grep -q "listening" "$LOGS/backend.log" || die "backend failed to start (see $LOGS/backend.log)"
ok "Backend up (NATS=$HOST)"

cleanup() { say "Stopping backend"; kill "$BACKEND_PID" 2>/dev/null || true; }
trap cleanup EXIT

# --- 4) Frontend (foreground) ----------------------------------------------
[ -d "$ROOT/ui/node_modules" ] || ( say "Installing UI deps"; cd "$ROOT/ui" && npm install )
ok "Frontend → http://localhost:$UI_PORT   (Ctrl+C stops backend + UI; nodes & Docker keep running)"
cd "$ROOT/ui" && npm run dev -- --port "$UI_PORT"
