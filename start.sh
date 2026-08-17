#!/usr/bin/env bash
#
# Start the Go backend + Vite UI against an ALREADY-RUNNING mpcium cluster.
#
# Starting mpcium (NATS, Consul, the 3 nodes) is on you. This script only
# checks the cluster is reachable, then boots backend + UI. All connection
# settings come from backend/config.yaml — the single source of truth.
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
HEALTH_BASE_PORT="$(cfg health_base_port)"
ADDR="$(cfg addr)"                                 # :8090
UI_PORT="${UI_PORT:-5173}"
HEALTH_HOST="${HEALTH_HOST:-127.0.0.1}"

CONSUL_HOST="${CONSUL_ADDR%%:*}"; CONSUL_PORT="${CONSUL_ADDR##*:}"
NATS_HOSTPORT="${NATS_URL#nats://}"
NATS_HOST="${NATS_HOSTPORT%%:*}"; NATS_PORT="${NATS_HOSTPORT##*:}"

reachable() { nc -z -w2 "$1" "$2" >/dev/null 2>&1; }

# --- 1) Check the mpcium cluster is up --------------------------------------
say "Checking mpcium cluster (NATS $NATS_HOST:$NATS_PORT, Consul $CONSUL_ADDR)"

hint() {
  cat <<EOF

  The mpcium cluster isn't ready. Start it yourself, then re-run ./start.sh:

    for n in 0 1 2; do ( cd $ROOT/mpcium/node\$n && mpcium start -n node\$n \\
      > $LOGS/node\$n.log 2>&1 & ); done
    curl -s http://$HEALTH_HOST:$HEALTH_BASE_PORT/health

  Connection targets live in $CFG.
EOF
}

reachable "$NATS_HOST" "$NATS_PORT"     || { warn "NATS unreachable at $NATS_HOST:$NATS_PORT"; hint; exit 1; }
reachable "$CONSUL_HOST" "$CONSUL_PORT" || { warn "Consul unreachable at $CONSUL_ADDR";        hint; exit 1; }
[ -n "$(curl -s "http://$CONSUL_ADDR/v1/status/leader" | tr -d '"')" ] \
  || { warn "Consul has no leader"; hint; exit 1; }
curl -s "http://$CONSUL_ADDR/v1/kv/mpc_peers/?keys" | grep -q node0 \
  || { warn "no peers registered in Consul (mpc_peers/ empty)"; hint; exit 1; }

live=0
for n in 0 1 2; do
  curl -s -m 2 "http://$HEALTH_HOST:$((HEALTH_BASE_PORT + n))/health" | grep -q '"live":true' \
    && live=$((live + 1))
done
[ "$live" -gt 0 ] || { warn "no mpcium nodes responding on $HEALTH_HOST:$HEALTH_BASE_PORT-$((HEALTH_BASE_PORT + 2))"; hint; exit 1; }
[ "$live" -eq 3 ] || warn "only $live/3 nodes live — keygen/signing may stall (2-of-3)"
ok "Cluster reachable ($live/3 nodes live)"

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
ok "Frontend → http://localhost:$UI_PORT   (Ctrl+C stops the UI; backend & mpcium keep running)"
cd "$ROOT/ui" && npm run dev -- --port "$UI_PORT"
