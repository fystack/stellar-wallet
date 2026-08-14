#!/usr/bin/env bash
#
# Bootstrap a fresh mpcium MPC cluster from nothing:
#   • installs mpcium + mpcium-cli (if missing)
#   • generates peers, node identities, and the event-initiator key
#   • writes per-node configs (chain_code, threshold, health ports)
#   • registers peers into Consul
#
# Idempotent: does nothing if the cluster is already set up.
# Requires a reachable NATS + Consul (start.sh brings up a local one via Docker).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
MPCIUM="${MPCIUM_DIR:-$ROOT/mpcium}"
HOST="${HOST:-127.0.0.1}"
NATS_PORT="${NATS_PORT:-4222}"
CONSUL_PORT="${CONSUL_PORT:-8500}"
HEALTH_BASE="${HEALTH_BASE:-8091}"
N="${NODES:-3}"

say() { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
die() { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

# Portable in-place sed.
sedi() { if sed --version >/dev/null 2>&1; then sed -i "$@"; else sed -i '' "$@"; fi; }

command -v go >/dev/null || die "go not found in PATH"
command -v openssl >/dev/null || die "openssl not found"

# --- 1) Ensure binaries ---
if ! command -v mpcium >/dev/null; then
  say "Installing mpcium"
  go install github.com/fystack/mpcium/cmd/mpcium@latest
fi
if ! command -v mpcium-cli >/dev/null; then
  say "Installing mpcium-cli"
  go install github.com/fystack/mpcium/cmd/mpcium-cli@latest
fi
export PATH="$(go env GOPATH)/bin:$PATH"

# --- 2) Skip if already set up ---
if [ -f "$MPCIUM/node0/config.yaml" ]; then
  say "Cluster already set up at $MPCIUM — skipping bootstrap"
  exit 0
fi

say "Bootstrapping a fresh cluster at $MPCIUM (host=$HOST)"
mkdir -p "$MPCIUM"
cd "$MPCIUM"

CHAIN_CODE="$(openssl rand -hex 32)"
BADGER_PW="$(openssl rand -hex 12)"

cat > config.yaml <<EOF
nats:
  url: nats://$HOST:$NATS_PORT
consul:
  address: $HOST:$CONSUL_PORT
mpc_threshold: 2
environment: development
badger_password: "$BADGER_PW"
event_initiator_algorithm: "ed25519"
event_initiator_pubkey: ""
chain_code: "$CHAIN_CODE"
db_path: "."
backup_enabled: true
backup_period_seconds: 300
backup_dir: backups
max_concurrent_keygen: 2
max_concurrent_signing: 10
healthcheck:
  enabled: true
  address: "0.0.0.0:$HEALTH_BASE"
EOF

# --- 3) Peers + event initiator ---
say "Generating peers"
mpcium-cli generate-peers -n "$N" >/dev/null

say "Generating event initiator"
mpcium-cli generate-initiator >/dev/null
PUB="$(grep -o '"public_key": *"[^"]*"' event_initiator.identity.json | cut -d '"' -f4)"
[ -n "$PUB" ] || die "could not read event_initiator public key"
sedi -E "s|^([[:space:]]*event_initiator_pubkey:).*|\\1 \"$PUB\"|" config.yaml

# --- 4) Per-node dirs, configs, identities ---
for i in $(seq 0 $((N - 1))); do
  say "Setting up node$i"
  mkdir -p "node$i/identity"
  port=$((HEALTH_BASE + i))
  sed "s|address: \"0.0.0.0:$HEALTH_BASE\"|address: \"0.0.0.0:$port\"|" config.yaml > "node$i/config.yaml"
  cp peers.json "node$i/"
  ( cd "node$i" && mpcium-cli generate-identity --node "node$i" >/dev/null )
done

# Distribute each node's public identity file to all the others.
for i in $(seq 0 $((N - 1))); do
  for j in $(seq 0 $((N - 1))); do
    [ "$i" != "$j" ] && cp -f "node$i/identity/node${i}_identity.json" "node$j/identity/"
  done
done

# --- 5) Register peers into Consul (wait for a leader first) ---
say "Waiting for Consul leader"
for i in $(seq 1 30); do
  [ -n "$(curl -s "http://$HOST:$CONSUL_PORT/v1/status/leader" | tr -d '"')" ] && break
  sleep 1
  [ "$i" = 30 ] && die "Consul at $HOST:$CONSUL_PORT has no leader"
done
say "Registering peers into Consul"
mpcium-cli register-peers >/dev/null

say "Bootstrap complete: $N nodes, initiator, peers registered"
