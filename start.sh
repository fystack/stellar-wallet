#!/usr/bin/env bash
#
# Start the Go backend + Vite UI against an already-running mpcium cluster.
#
# The mpcium cluster (NATS, Consul, the nodes) is started and operated
# separately. This script only checks NATS + Consul are reachable, then boots
# backend + UI. All connection settings come from backend/config.yaml — the
# single source of truth.
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
LOGS="$ROOT/logs"
CFG="$ROOT/backend/config.yaml"
mkdir -p "$LOGS"

say()  { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
ok()   { printf "\033[1;32m✓ %s\033[0m\n" "$1"; }
warn() { printf "\033[1;33m! %s\033[0m\n" "$1"; }
die()  { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

command -v go  >/dev/null || die "go not found"
command -v npm >/dev/null || die "npm not found"
[ -f "$CFG" ] || die "missing $CFG (copy backend/config.example.yaml)"

# Read a top-level scalar from config.yaml (strips inline comments + quotes).
cfg() {
  sed -nE "s|^$1:[[:space:]]*(.*)\$|\1|p" "$CFG" | head -1 \
    | sed -E 's/[[:space:]]+#.*$//; s/^"//; s/"$//; s/[[:space:]]*$//'
}

CONSUL_ADDR="$(cfg consul_addr)"                  # host:port
NATS_URL="$(cfg nats_url)"                         # nats://host:port
ADDR="$(cfg addr)"                                 # :8090
UI_PORT="${UI_PORT:-5173}"

CONSUL_HOST="${CONSUL_ADDR%%:*}"; CONSUL_PORT="${CONSUL_ADDR##*:}"
NATS_HOSTPORT="${NATS_URL#nats://}"
NATS_HOST="${NATS_HOSTPORT%%:*}"; NATS_PORT="${NATS_HOSTPORT##*:}"

reachable() { nc -z -w2 "$1" "$2" >/dev/null 2>&1; }

# --- 1) Check NATS + Consul are reachable -----------------------------------
# The mpcium cluster is started and operated separately; we only need its
# NATS + Consul endpoints to be up so the backend can connect.
say "Checking backend deps (NATS $NATS_HOST:$NATS_PORT, Consul $CONSUL_ADDR)"

hint() {
  cat <<EOF

  NATS/Consul aren't reachable. Make sure the mpcium cluster is up, then
  re-run ./start.sh. Connection targets live in $CFG.
EOF
}

reachable "$NATS_HOST" "$NATS_PORT"     || { warn "NATS unreachable at $NATS_HOST:$NATS_PORT"; hint; exit 1; }
reachable "$CONSUL_HOST" "$CONSUL_PORT" || { warn "Consul unreachable at $CONSUL_ADDR";        hint; exit 1; }
[ -n "$(curl -s "http://$CONSUL_ADDR/v1/status/leader" | tr -d '"')" ] \
  || { warn "Consul has no leader"; hint; exit 1; }
ok "NATS + Consul reachable"

# --- 2) Backend (reads backend/config.yaml) ---------------------------------
say "Building & starting backend on $ADDR"
pkill -f "backend/stellar-wallet-backend" 2>/dev/null || true
( cd "$ROOT/backend" && go build -o stellar-wallet-backend . ) || die "backend build failed"
( cd "$ROOT/backend" && ./stellar-wallet-backend > "$LOGS/backend.log" 2>&1 & )
sleep 2
grep -q "listening" "$LOGS/backend.log" || die "backend failed to start (see $LOGS/backend.log)"
ok "Backend up on $ADDR"

# --- 3) Frontend (foreground) ----------------------------------------------
[ -d "$ROOT/ui/node_modules" ] || ( say "Installing UI deps"; cd "$ROOT/ui" && npm install )
ok "Frontend → http://localhost:$UI_PORT   (Ctrl+C stops the UI; backend keeps running)"
cd "$ROOT/ui" && npm run dev -- --port "$UI_PORT"
