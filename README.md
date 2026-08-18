# Stellar MPC Wallet

A threshold-signature (2-of-3 MPC) wallet for Stellar & Solana, powered by the
[mpcium](https://github.com/fystack/mpcium) signing cluster. The private key is
generated in shares across 3 nodes and **never assembled** — every transaction
is signed by 2 of 3 nodes.

```
stellar-wallet/
├── backend/     # Go (Gin) API — links to the cluster over NATS, builds/broadcasts txs
├── ui/          # Vite + React + Tailwind frontend
├── start.sh     # start backend + UI
└── stop.sh      # stop backend + UI
```

This repo is the **webapp only** (backend + UI). The mpcium cluster (NATS,
Consul, the nodes) is started and operated **separately** — see the
[mpcium](https://github.com/fystack/mpcium) project. The backend connects to it
over the NATS/Consul endpoints configured in `backend/config.yaml`.

## Prerequisites

- **Go** ≥ 1.21, **Node** ≥ 18 / npm.
- A **running mpcium cluster** reachable at the NATS + Consul addresses in
  `backend/config.yaml` (dev default: `10.10.0.1:4222` / `10.10.0.1:8500`).

## Run

```bash
./start.sh
```

`start.sh` checks that NATS + Consul are reachable, then builds & starts the
backend on **:8090** and the frontend on **:5173**. All connection settings come
from `backend/config.yaml` (the single source of truth).

Open **http://localhost:5173**, register, and create a wallet.

Stop:

```bash
./stop.sh            # backend + UI (the mpcium cluster is untouched)
```

Logs are written to `logs/` (`backend.log`).

## Configuration (env)

| Var            | Default            | Used by     |
| -------------- | ------------------ | ----------- |
| `UI_PORT`      | `5173`             | start.sh    |
| `VITE_API_BASE`| `http://localhost:8090` | ui     |

Backend connection settings (`addr`, `nats_url`, `consul_addr`, `db_path`,
event-initiator key, …) live in `backend/config.yaml`.

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
