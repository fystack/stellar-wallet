#!/usr/bin/env bash
#
# Generate the mpcium node identities, peers.json and event-initiator key that
# the Docker stack mounts — so a fresh clone can `docker-compose up` with no
# manual config. Idempotent: if the keys already exist it does nothing.
#
# Uses the official mpcium-cli image (no local Go toolchain needed). Runs the
# CLI as your host user so the generated files are yours to edit; the node
# containers run as root and can read them regardless.
#
#   ./scripts/gen-keys.sh            # generate if missing
#   FORCE=1 ./scripts/gen-keys.sh    # wipe and regenerate (loses key-shares!)
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MPC_DIR="${MPC_DIR:-$ROOT/mpcium}"
CFG="${CFG:-$ROOT/infra/mpcium.docker.yaml}"
CLI_IMAGE="${CLI_IMAGE:-docker.io/fystacklabs/mpcium-cli:0.3.5}"
NODES="${NODES:-3}"

say()  { printf "\033[1;34m▶ %s\033[0m\n" "$1"; }
ok()   { printf "\033[1;32m✓ %s\033[0m\n" "$1"; }
die()  { printf "\033[1;31m✖ %s\033[0m\n" "$1"; exit 1; }

# Portable in-place sed (BSD/macOS vs GNU).
sedi() { if sed --version >/dev/null 2>&1; then sed -i "$@"; else sed -i '' "$@"; fi; }

command -v docker >/dev/null || die "docker not found"
[ -f "$CFG" ] || die "missing $CFG"

if [ "${FORCE:-0}" = "1" ]; then
  say "FORCE=1 — removing existing $MPC_DIR/node* + keys"
  docker run --rm -v "$MPC_DIR:/data" -w /data alpine sh -c 'rm -rf node* peers.json event_initiator.*' 2>/dev/null || true
fi

if [ -f "$MPC_DIR/node0/identity/node0_private.key" ]; then
  ok "keys already present in $MPC_DIR — nothing to do (FORCE=1 to regenerate)"
  exit 0
fi

mkdir -p "$MPC_DIR"

# Run mpcium-cli in the given dir (mounted at /data), as the host user.
run_cli() {
  local dir="$1"; shift
  docker run --rm --user "$(id -u):$(id -g)" -e USER=nonroot \
    -v "$dir:/data" -w /data "$CLI_IMAGE" "$@"
}

say "Pulling $CLI_IMAGE"
docker pull "$CLI_IMAGE" >/dev/null || die "cannot pull $CLI_IMAGE"

# 1) peers.json (fresh random node IDs)
say "Generating peers.json ($NODES nodes)"
run_cli "$MPC_DIR" generate-peers -n "$NODES" -o peers.json
[ -f "$MPC_DIR/peers.json" ] || die "peers.json not generated"

# 2) per-node identity (each in its own dir; needs peers.json alongside)
for i in $(seq 0 $((NODES - 1))); do
  node="node$i"; dir="$MPC_DIR/$node"
  say "Generating identity for $node"
  mkdir -p "$dir/identity"
  cp "$MPC_DIR/peers.json" "$dir/peers.json"
  run_cli "$dir" generate-identity --node "$node"
done

# 3) distribute each node's public identity to every other node
say "Distributing identity files"
for i in $(seq 0 $((NODES - 1))); do
  src="$MPC_DIR/node$i/identity/node${i}_identity.json"
  [ -f "$src" ] || die "missing $src"
  for j in $(seq 0 $((NODES - 1))); do
    [ "$i" -ne "$j" ] && cp "$src" "$MPC_DIR/node$j/identity/"
  done
done

# 4) event initiator key (backend signs keygen/sign requests with this)
say "Generating event initiator key"
run_cli "$MPC_DIR" generate-initiator
[ -f "$MPC_DIR/event_initiator.identity.json" ] || die "event_initiator not generated"

# 5) inject the initiator public key into the node config
PUBKEY="$(python3 -c "import json;print(json.load(open('$MPC_DIR/event_initiator.identity.json'))['public_key'])")"
[ -n "$PUBKEY" ] || die "could not read initiator public_key"
sedi "s|event_initiator_pubkey:.*|event_initiator_pubkey: \"$PUBKEY\"|" "$CFG"
ok "Set event_initiator_pubkey in $(basename "$CFG")"

# readable by the containers
chmod -R a+r "$MPC_DIR"/node*/identity "$MPC_DIR"/node*/peers.json 2>/dev/null || true

ok "Keys generated in $MPC_DIR — you can now run: docker-compose up -d"
