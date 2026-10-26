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
  apperror/                  # Business errors and wrapped causes
  httpapi/                   # JSON envelopes and HTTP error mapping
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
their service/repository extraction can happen incrementally. REST paths and the
SQLite schema remain unchanged. JSON envelopes follow the contract below.

## JSON response contract

All ordinary JSON endpoints, including `/health`, return a success envelope:

```json
{"data": {"id": "wallet-1"}}
```

Lists remain arrays inside `data`. Optional pagination metadata belongs in `meta`:

```json
{"data": [], "meta": {"next_cursor": null, "has_more": false}}
```

Metadata is omitted when unused; this change does not add pagination to existing
endpoints. Errors use a separate envelope with no `data` field:

```json
{"error": {"code": "wallet_not_found", "message": "Wallet not found."}}
```

Clients branch on `error.code`, not the message. `ui/src/api.ts` unwraps `data`
and exposes `ApiError.status`, `.code`, and `.message`. Callers still receive the
same wallet/transaction objects and arrays. Older clients consuming unwrapped
JSON must migrate with this change; no legacy-response fallback is provided.

- Resource creation (register, wallet, payment, swap, trustline): `201`.
- Reads, login and other successful actions: `200`.
- Invalid input: `400`; authentication: `401`; permissions: `403`.
- Missing resource/route: `404`; unsupported method: `405`; conflict: `409`.
- Dependency failure: `502`; insufficient MPC availability: `503`.
- Unexpected failures: `500` with `internal_error` and a generic message.
- `204` and CORS preflight responses have no body.
- SSE keeps its `data: {"type": ..., "data": ...}` event format. Authentication
  failures before the stream opens use the JSON error envelope.
- OAuth redirects, when implemented, will remain redirects rather than JSON.

`apperror.Error` carries a transport-independent kind, stable code, public message
and optional cause. Services can wrap technical causes with `%w`; `httpapi.Error`
uses `errors.As` to map failures to HTTP and logs technical causes server-side.
Unexpected SQL, SDK or provider details are never copied into HTTP error messages.
Repositories use `repository.ErrNotFound` / `ErrConflict` for known failures and
preserve underlying errors when wrapping. Existing legacy lookup/update methods
that do not return errors retain their current signatures for now.

## Verify

```bash
go test ./...
go vet ./...
go build ./...
```

Frontend API client tests use the built-in Node test runner (no added dependency):

```bash
cd ../ui
npm test
npm run build
```

## Endpoints (`/api/v1`)

| Method | Path                          | Auth | Body / notes                          |
| ------ | ----------------------------- | ---- | ------------------------------------- |
| POST   | `/auth/register`              | –    | `{email, password}` → `data.access_token`|
| POST   | `/auth/login`                 | –    | `{email, password}` → `data.access_token`|
| GET    | `/wallets`                    | ✓    | list wallets                          |
| POST   | `/wallets`                    | ✓    | `{name, chain}` (`stellar`)           |
| GET    | `/wallets/:id`                | ✓    | one wallet                            |
| GET    | `/wallets/:id/transactions`   | ✓    | wallet transactions                   |
| POST   | `/transactions`               | ✓    | `{wallet_id, to, amount, memo}`       |
| GET    | `/transactions/:id`           | ✓    | one transaction                       |
| GET    | `/cluster`                    | –    | real MPC peer/node status             |

Outgoing Stellar transactions move through `signing → broadcast → confirmed`
from real MPC result events.
