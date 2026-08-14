# Stellar Wallet Backend

Go (Gin) + SQLite API for the Stellar/Solana wallet. Key generation and signing
use the real mpcium cluster over NATS; transaction and wallet updates are pushed
to clients over SSE.

## Run

The normal entrypoint is the root setup script, which starts NATS, Consul, the
MPC nodes, this backend, and the UI:

```bash
./start.sh
```

To run only the backend against an existing cluster:

```bash
cd backend
ADDR=:8090 \
NATS_URL=nats://127.0.0.1:14222 \
CONSUL_ADDR=127.0.0.1:18500 \
INITIATOR_KEY=../mpcium/event_initiator.key \
go run .
```

## Architecture

`main.go` is only the composition root. Runtime responsibilities live under
`internal/` with one-way, acyclic dependencies:

```text
main.go
  ├── app/       HTTP routes, handlers, use-case orchestration, MPC callbacks
  ├── auth/      JWT creation and authentication middleware
  ├── chain/     Stellar/Solana RPC, transaction building, prices
  ├── cluster/   Consul peer discovery and MPC node health
  ├── domain/    Wallet, transaction, asset, and cluster models
  ├── mpcium/    NATS/mpcium client adapter
  ├── realtime/  Per-user SSE event hub
  ├── store/     Database-neutral persistence contract
  └── storage/
      └── sqlite/  Current schema and repository adapter
```

The app layer depends only on `store.Store`, not SQL or a database driver. All
SQLite-specific schema and queries are contained in `internal/storage/sqlite`.
To migrate databases, implement the same port in an adapter such as
`internal/storage/postgres` and change only the wiring in `main.go`; handlers,
use cases, routes, and domain models stay unchanged. Public REST paths, JSON
shapes, and the current SQLite schema remain unchanged by this package split.

## Verify

```bash
go test ./...
go vet ./...
go build ./...
```

## Endpoints (`/api/v1`)

| Method | Path                          | Auth | Body / notes                          |
| ------ | ----------------------------- | ---- | ------------------------------------- |
| POST   | `/auth/register`              | –    | `{email, password}` → `{access_token}`|
| POST   | `/auth/login`                 | –    | `{email, password}` → `{access_token}`|
| GET    | `/wallets`                    | ✓    | list wallets                          |
| POST   | `/wallets`                    | ✓    | `{name, chain}` (`stellar` or `solana`) |
| GET    | `/wallets/:id`                | ✓    | one wallet                            |
| GET    | `/wallets/:id/transactions`   | ✓    | wallet transactions                   |
| POST   | `/transactions`               | ✓    | `{wallet_id, to, amount, memo}`       |
| GET    | `/transactions/:id`           | ✓    | one transaction                       |
| GET    | `/cluster`                    | –    | real MPC peer/node status             |

Outgoing Stellar transactions move through `signing → broadcast → confirmed`
from real MPC result events. Solana currently stops after MPC signing and does
not broadcast on-chain.
