#!/usr/bin/env bash
#
# Stop the backend and UI.
#
# The mpcium cluster is started and operated separately — this script does
# not touch it.
#
set -uo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "Stopping backend + UI…"
pkill -f "stellar-wallet-backend" 2>/dev/null || true
pkill -f "vite" 2>/dev/null || true
rm -f "$ROOT/logs/backend.pid" 2>/dev/null || true

echo "Done."
