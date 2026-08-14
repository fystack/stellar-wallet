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

- **Go** ≥ 1.21, **Node** ≥ 18 / npm.
- **Docker** + docker compose (plugin *or* `docker-compose`) — for local NATS/Consul.
- `mpcium` / `mpcium-cli` are **auto-installed** by `bootstrap.sh` if missing
  (`go install github.com/fystack/mpcium/cmd/...`). The mpcium repo itself is never needed.

The `mpcium/` folder (node data + keys) is **not in this repo** — it's created on first run.

## Run

```bash
./start.sh
```

It's **seamless** — you don't set up NATS/Consul, keys, or nodes yourself:

1. **No `mpcium/` folder** (fresh clone) → `bootstrap.sh` runs: installs mpcium/mpcium-cli,
   boots local NATS+Consul (Docker), generates peers + node identities + the event-initiator
   key, writes per-node configs, and registers peers into Consul.
2. Else if the shared dev cluster (`10.10.0.1:4222/8500`) is reachable → use it.
3. Else → reuse the existing local setup (bringing Docker infra up if needed).
4. Starts the 3 MPC nodes (only if not already running — never duplicates).
5. Builds & starts the backend on **:8090**, pointed at the chosen infra.
6. Installs UI deps if needed and starts the frontend on **:5173**.

Open **http://localhost:5173**, register, and create a wallet.

If `4222`/`8500` are busy on your machine, override the ports:

```bash
NATS_PORT=14222 CONSUL_PORT=18500 ./start.sh
```

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
