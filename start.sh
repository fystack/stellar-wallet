#!/usr/bin/env bash
#
# Seamless start: MPC cluster (NATS+Consul+3 nodes) + Go backend + Vite UI.
#
# Picks infra automatically:
#   • if the shared dev NATS/Consul (10.10.0.1) is reachable → use it
#   • otherwise → boot a LOCAL NATS + Consul via Docker and register peers
#
# mpcium is used as installed binaries (mpcium / mpcium-cli). No repo build needed.
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
MPCIUM="$ROOT/mpcium"
LOGS="$ROOT/logs"
mkdir -p "$LOGS"

SHARED_HOST="${SHARED_HOST:-10.10.0.1}"
BACKEND_ADDR="${BACKEND_ADDR:-:8090}"
UI_PORT="${UI_PORT:-5173}"

say()  { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
ok()   { printf "\033[1;32m✓ %s\033[0m\n" "$1"; }
die()  { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

command -v mpcium >/dev/null || die "mpcium not found (go install github.com/fystack/mpcium/cmd/mpcium@latest)"
command -v go  >/dev/null || die "go not found"
command -v npm >/dev/null || die "npm not found"

reachable() { nc -z -w2 "$1" "$2" >/dev/null 2>&1; }

# --- 1) Decide infra host ---------------------------------------------------
if reachable "$SHARED_HOST" 4222 && reachable "$SHARED_HOST" 8500; then
  HOST="$SHARED_HOST"
  ok "Using shared cluster at $HOST"
else
  say "Shared cluster unreachable — booting local NATS + Consul (Docker)"
  command -v docker >/dev/null || die "docker not found (needed for local infra)"
  docker compose -f "$MPCIUM/docker-compose.yaml" up -d >/dev/null 2>&1 \
    || die "docker compose up failed"
  HOST="127.0.0.1"
  say "Waiting for local NATS/Consul…"
  for i in $(seq 1 30); do
    reachable "$HOST" 4222 && reachable "$HOST" 8500 && break
    sleep 1
    [ "$i" = 30 ] && die "local NATS/Consul did not come up"
  done
  ok "Local NATS/Consul ready"
fi

# --- 2) Point mpcium configs at the chosen host -----------------------------
align_host() {
  local f="$1"
  [ -f "$f" ] || return 0
  sed -i '' -E "s|url: nats://[^:]+:4222|url: nats://$HOST:4222|" "$f" 2>/dev/null || \
  sed -i    -E "s|url: nats://[^:]+:4222|url: nats://$HOST:4222|" "$f"
  sed -i '' -E "s|address: [0-9a-zA-Z.]+:8500|address: $HOST:8500|" "$f" 2>/dev/null || \
  sed -i    -E "s|address: [0-9a-zA-Z.]+:8500|address: $HOST:8500|" "$f"
}
align_host "$MPCIUM/config.yaml"
for n in 0 1 2; do align_host "$MPCIUM/node$n/config.yaml"; done

# --- 3) Register peers if Consul has none (fresh local cluster) -------------
if ! curl -s "http://$HOST:8500/v1/kv/mpc_peers/?keys" | grep -q node0; then
  say "Registering peers into Consul"
  command -v mpcium-cli >/dev/null || die "mpcium-cli not found (needed to register peers)"
  ( cd "$MPCIUM" && mpcium-cli register-peers ) || die "register-peers failed"
fi

# --- 4) Start the 3 MPC nodes (only if not already running) -----------------
for n in 0 1 2; do
  if pgrep -f "mpcium start -n node$n" >/dev/null; then
    say "node$n already running"
  else
    say "Starting node$n"
    ( cd "$MPCIUM/node$n" && mpcium start -n "node$n" > "$LOGS/node$n.log" 2>&1 & )
  fi
done
sleep 4

# --- 5) Backend -------------------------------------------------------------
say "Building & starting backend on $BACKEND_ADDR"
( cd "$ROOT/backend" && go build -o stellar-wallet-backend . ) || die "backend build failed"
ADDR="$BACKEND_ADDR" NATS_URL="nats://$HOST:4222" CONSUL_ADDR="$HOST:8500" \
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

# --- 6) Frontend (foreground) ----------------------------------------------
[ -d "$ROOT/ui/node_modules" ] || ( say "Installing UI deps"; cd "$ROOT/ui" && npm install )
ok "Frontend → http://localhost:$UI_PORT   (Ctrl+C stops backend + UI; nodes & Docker keep running)"
cd "$ROOT/ui" && npm run dev -- --port "$UI_PORT"
