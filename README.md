# Stellar MPC Wallet

A threshold-signature (2-of-3 MPC) wallet for Stellar, powered by the
[mpcium](https://github.com/fystack/mpcium) signing cluster. The private key is
generated in shares across 3 nodes and **never assembled** — every transaction
is signed by 2 of 3 nodes.

```
stellar-wallet/
├── backend/              # Go (Gin) API — links to the cluster over NATS, builds/broadcasts txs
├── ui/                   # Vite + React + Tailwind frontend
├── mpcium/               # node identities + BadgerDB key-shares (mounted into the node containers)
├── infra/                # mpcium Dockerfile + node config for the Docker stack
├── docker-compose.yaml   # full stack: NATS + Consul + 3 nodes + backend + UI
├── start.sh / stop.sh    # local dev (backend + UI against an external cluster)
```

## Run with Docker (everything)

Brings up NATS, Consul, the 3 mpcium nodes, the backend, and the UI — then you
just open the web:

```bash
docker compose up --build -d
```

Open **http://localhost:8080**, register, and create a wallet. That's it.

- UI on **:8080** (nginx; proxies `/api` + SSE to the backend, so single-origin)
- Backend on **:8090**, mpcium node health on **:8091–8093**, Consul on **:8500**
- The nodes reuse the existing key-shares under `mpcium/nodeN/` (bind-mounted);
  Consul runs in dev mode and each node re-seeds its peer IDs on startup.
- Wallet data lives in the `backend_data` volume (fresh DB on first run).

```bash
docker compose logs -f backend        # follow a service
docker compose down                   # stop everything (key-shares persist on host)
docker compose down -v                # also wipe the wallet DB volume
```

## Run locally (backend + UI only, external cluster)

For iterating on the app against an already-running mpcium cluster (e.g. the
shared dev cluster), skip Docker:

```bash
./start.sh   # checks NATS+Consul reachable, then backend :8090 + UI :5173
./stop.sh    # stops backend + UI (the cluster is untouched)
```

Connection settings come from `backend/config.yaml` (the single source of
truth). Logs are written to `logs/`.

## Prerequisites

- **Docker** + compose (for the full-stack path), or
- **Go** ≥ 1.21, **Node** ≥ 18 / npm (for the local path).

## Configuration (env)

| Var            | Default            | Used by     |
| -------------- | ------------------ | ----------- |
| `UI_PORT`      | `5173`             | start.sh    |
| `VITE_API_BASE`| `http://localhost:8090` | ui     |

Backend connection settings (`addr`, `nats_url`, `consul_addr`, `db_path`,
event-initiator key, …) live in `backend/config.yaml`.

The Stellar Horizon RPC endpoint is editable at runtime in **Settings → Chains & RPC**.

## What works

- **Keygen** — real distributed MPC keygen; Stellar (`G…`) address derived from the EdDSA pubkey.
- **Send** — builds a Stellar payment / createAccount, signs via the cluster, broadcasts to Horizon (testnet). Address + balance validation, fee estimate, memo on-chain.
- **Receive** — QR + polls Horizon for incoming payments.
- **Balances** — live per-asset, cached with a background refresher, USD values (CoinGecko).
- **Custom assets** — register an asset (code+issuer) in Settings, add a trustline per wallet.
- **Cluster health** — real node liveness via each node's `/health`; RPC status checks.
- Live updates over SSE, hash routing (deep-links / reload-safe), toasts.

## Notes / limits

- **Testnet only** — Horizon testnet + Friendbot. Mainnet not enabled.
- Dev auth (hardcoded JWT secret). Harden (env secret, HTTPS, rate-limit) before any real deployment.

## Backend API (`/api/v1`)

`auth/register`, `auth/login`, `wallets` (CRUD), `wallets/:id/{balance,fund,sync,trustline,transactions}`,
`transactions`, `events` (SSE), `cluster`, `chains`, `prices`, `config`, `assets`.
