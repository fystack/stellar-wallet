# Stellar MPC Wallet

A threshold-signature (2-of-3 MPC) wallet for Stellar & Solana, powered by the
[mpcium](https://github.com/fystack/mpcium) signing cluster. The private key is
generated in shares across 3 nodes and **never assembled** — every transaction
is signed by 2 of 3 nodes.

```
stellar-wallet/
├── mpcium/      # the MPC cluster (node0/1/2 data + identities). Runs via the `mpcium` binary.
├── backend/     # Go (Gin) API — links to the cluster over NATS, builds/broadcasts txs
├── ui/          # Vite + React + Tailwind frontend
├── start.sh     # start everything
└── stop.sh      # stop backend + UI (+ --nodes to stop the cluster)
```

## Prerequisites

- **mpcium** + **mpcium-cli** installed:
  `go install github.com/fystack/mpcium/cmd/mpcium@latest` (and `.../cmd/mpcium-cli@latest`).
  You do *not* need to build the mpcium repo — they're used as installed CLIs.
- **Go** ≥ 1.21, **Node** ≥ 18 / npm.
- **Docker** — only needed for the local-infra fallback (see below).
- The `mpcium/node0|1|2` folders with their `identity/` + `config.yaml` (already set up here).

## Run

```bash
./start.sh
```

It's **seamless** about infra — you don't have to start NATS/Consul yourself:

1. If the shared dev cluster (`10.10.0.1:4222/8500`) is reachable → use it.
2. Otherwise → boots a **local NATS + Consul via Docker**, points the node configs at
   `127.0.0.1`, and registers the peers into Consul automatically.
3. Starts the 3 MPC nodes (only if not already running — never duplicates).
4. Builds & starts the backend on **:8090** (8080 is taken by other services here),
   pointed at whichever infra was chosen.
5. Installs UI deps if needed and starts the frontend on **:5173**.

Open **http://localhost:5173**, register, and create a wallet.

Stop:

```bash
./stop.sh            # backend + UI (nodes keep running)
./stop.sh --nodes    # also stop the MPC node processes
./stop.sh --all      # also stop the local Docker NATS/Consul
```

Logs are written to `logs/` (`node0.log`, `backend.log`, …).

## Configuration (env)

| Var            | Default            | Used by     |
| -------------- | ------------------ | ----------- |
| `BACKEND_ADDR` | `:8090`            | start.sh    |
| `UI_PORT`      | `5173`             | start.sh    |
| `NATS_HOST`    | `10.10.0.1`        | start.sh    |
| `NATS_URL`     | `nats://10.10.0.1:4222` | backend |
| `CONSUL_ADDR`  | `10.10.0.1:8500`   | backend     |
| `INITIATOR_KEY`| `../mpcium/event_initiator.key` | backend |
| `DB_PATH`      | `wallet.db`        | backend     |
| `VITE_API_BASE`| `http://localhost:8090` | ui     |

RPC endpoints (Stellar Horizon, Solana) are editable at runtime in **Settings → Chains & RPC**.

## What works

- **Keygen** — real distributed MPC keygen; Stellar (`G…`) & Solana addresses derived from the EdDSA pubkey.
- **Send** — builds a Stellar payment / createAccount, signs via the cluster, broadcasts to Horizon (testnet). Address + balance validation, fee estimate, memo on-chain.
- **Receive** — QR + polls Horizon for incoming payments.
- **Balances** — live per-asset, cached with a background refresher, USD values (CoinGecko).
- **Custom assets** — register an asset (code+issuer) in Settings, add a trustline per wallet.
- **Cluster health** — real node liveness via each node's `/health`; RPC status checks.
- Live updates over SSE, hash routing (deep-links / reload-safe), toasts.

## Notes / limits

- **Testnet only** — Horizon testnet + Friendbot, Solana devnet + airdrop. Mainnet not enabled.
- **Solana** — keygen, address, balance, fund work; on-chain *send* is sign-only (no broadcast yet).
- Dev auth (hardcoded JWT secret). Harden (env secret, HTTPS, rate-limit) before any real deployment.

## Backend API (`/api/v1`)

`auth/register`, `auth/login`, `wallets` (CRUD), `wallets/:id/{balance,fund,sync,trustline,transactions}`,
`transactions`, `events` (SSE), `cluster`, `chains`, `prices`, `config`, `assets`.
