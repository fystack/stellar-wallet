# Stellar Wallet Backend

Go (Gin) + SQLite API for the Stellar wallet. Key generation and signing
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

`main.go` loads configuration and infrastructure. `app/` connects the services,
HTTP handlers and MPC callbacks, then starts runtime jobs.

```text
internal/
  app/
    server.go                # Wiring and startup only
  handler/                   # HTTP: bind requests, call services, return responses
    router.go
    auth.go
    wallet.go
    payment.go
    swap.go
    config.go
    events.go
  service/                   # Business logic and background work; no Gin/HTTP
    payment.go               # Create, sign, broadcast and recover payments
    wallet.go                # Keygen callbacks and incoming payment sync
    balances.go              # Balance refresh jobs
    mpc_policy.go            # Node quorum and keygen timeout/recovery
  repository/
    store.go                 # Existing user/wallet/settings persistence contract
    sqlite/
      payment_repository.go  # Concrete transaction repository
      store.go               # Existing user/wallet/settings SQL
      database.go            # Schema and database setup
  domain/                    # Data models
  auth/                      # JWT and auth middleware
  chain/                     # Stellar RPC, transaction builders and prices
  mpc/                       # mpcium/NATS adapter
  cluster/                   # MPC node health and discovery
  realtime/                  # Per-user SSE hub
```

Payment requests follow `handler/payment.go → service/payment.go →
repository/sqlite/payment_repository.go`. The payment handler binds JSON and maps
service errors to HTTP responses. `PaymentService` checks wallet ownership and
signing quorum, resolves the recipient, builds and persists the payment, and
dispatches MPC signing. It also owns signing timeouts, pending-payment recovery,
and the shared result/broadcast lifecycle used by sends, swaps, and trustlines.
`WalletService` refreshes balances after a successful broadcast.

The payment repository is concrete; interfaces cover the external Stellar,
cluster and signer boundaries used in tests. Existing auth, wallet creation,
swap, trustline and settings handlers still contain their original orchestration;
their service/repository extraction can happen incrementally. REST paths, JSON
shapes, and the SQLite schema remain unchanged.

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
| POST   | `/wallets`                    | ✓    | `{name, chain}` (`stellar`)           |
| GET    | `/wallets/:id`                | ✓    | one wallet                            |
| GET    | `/wallets/:id/transactions`   | ✓    | wallet transactions                   |
| POST   | `/transactions`               | ✓    | `{wallet_id, to, amount, memo}`       |
| GET    | `/transactions/:id`           | ✓    | one transaction                       |
| GET    | `/cluster`                    | –    | real MPC peer/node status             |

Outgoing Stellar transactions move through `signing → broadcast → confirmed`
from real MPC result events.
