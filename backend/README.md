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

To run only the backend against an existing cluster, copy
`config.example.yaml` to `config.yaml`, point `nats_url`, `consul_addr` and
`initiator_key` at the cluster, then:

```bash
cd backend
go run .
```

Settings come only from `config.yaml`; environment variables are not read.

## Architecture

`main.go` loads configuration and infrastructure. `app/` connects the services,
HTTP handlers and MPC callbacks, then starts runtime jobs.

```text
internal/
  app/
    server.go                # Wiring and startup only
  handler/                   # HTTP: bind requests, call services, return responses
    router.go                # Route table: wires every area handler in one place
    middleware.go            # Logging, panic recovery, CORS, CSRF
    auth.go                  # AuthHandler
    wallet.go                # WalletHandler (incl. trustlines)
    transaction.go           # TransactionHandler
    swap.go                  # SwapHandler
    settings.go              # SettingsHandler (config, assets, chains, prices, cluster)
    events.go                # EventsHandler (SSE)
  service/                   # Business logic and background work; no Gin/HTTP
    auth.go                  # Password/social login, linking and sessions
    wallet.go                # Wallet lifecycle, keygen callbacks, funding and incoming sync
    transaction.go           # Payments, transaction reads and the shared sign/broadcast lifecycle
    swap.go                  # DEX quotes and path-payment swaps
    trustline.go             # Trustline transactions
    settings.go              # Custom assets, chain status (Horizon URL from config) and prices
    balances.go              # Balance refresh jobs
    mpc_policy.go            # Node quorum and keygen/sign timeout recovery
    ownership.go             # Wallet ownership/readiness checks
  repository/
    store.go                 # Existing user/wallet/settings persistence contract
    sqlite/
      auth.go                # Accounts, identities, sessions and OAuth state
      transactions.go        # Concrete transaction repository
      store.go               # Existing user/wallet/settings SQL
      database.go            # Schema and database setup
  apperror/                  # Business errors and wrapped causes
  httpapi/                   # JSON envelopes and HTTP error mapping
  domain/                    # Data models
  auth/                      # OAuth/OIDC providers and cookie-session middleware
  chain/                     # Stellar RPC, transaction builders and prices
  mpc/                       # mpcium/NATS adapter
  cluster/                   # MPC node health and discovery
  realtime/                  # Per-user SSE hub
```

Payment requests follow `handler/transaction.go → service/transaction.go →
repository/sqlite/transactions.go`. The handler binds JSON and maps service
errors to HTTP responses. `TransactionService` checks wallet ownership and
signing quorum, resolves the recipient, builds and persists the payment, and
dispatches MPC signing. It also owns signing timeouts, pending-transaction
recovery, and the shared result/broadcast lifecycle used by sends, swaps and
trustlines. `WalletService` refreshes balances after a successful broadcast.

Each API area has its own small handler struct holding only the services it
uses; `router.go` builds them and lists every route. Every handler method
follows the same shape: bind the request, call one service method, write the
response. Swaps and trustlines build their own envelopes, then reuse
`TransactionService`'s dispatch to persist, sign and broadcast.

Wallet and transaction statuses and transaction types are typed constants in
`domain/status.go`. Repository writes return errors: request paths return them
to the client, and background paths (MPC callbacks, timeouts, balance refresh)
log them.

The repositories are concrete; interfaces cover the external Stellar
(`PaymentChain`, `SwapChain`, `TrustlineChain`), cluster and signer boundaries,
so services are tested with stubs. JSON envelopes follow the contract below.

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
- Reads, login and other successful actions: `200`. Logout/reauthentication: `204`.
- Invalid input: `400`; authentication: `401`; permissions: `403`.
- Missing resource/route: `404`; unsupported method: `405`; conflict: `409`.
- Dependency failure: `502`; insufficient MPC availability: `503`.
- Unexpected failures: `500` with `internal_error` and a generic message.
- `204` and CORS preflight responses have no body.
- SSE keeps its `data: {"type": ..., "data": ...}` event format. Authentication
  failures before the stream opens use the JSON error envelope.
- OAuth start/callback use redirects rather than JSON. See [OAuth setup](../docs/oauth.md).

`apperror.Error` carries a transport-independent kind, stable code, public message
and optional cause. Services can wrap technical causes with `%w`; `httpapi.Error`
uses `errors.As` to map failures to HTTP and logs technical causes server-side.
Unexpected SQL, SDK or provider details are never copied into HTTP error messages.
Repositories use `repository.ErrNotFound` / `ErrConflict` for known failures and
preserve underlying errors when wrapping. Existing legacy lookup/update methods
that do not return errors retain their current signatures for now.

## Social login and sessions

Google/GitHub OAuth, password login, explicit account linking and revocable
HttpOnly cookie sessions are supported. See [setup and reproduction](../docs/oauth.md)
for callback URLs, private configuration, migration and tests. Existing users keep
their user IDs and wallets; old JWT sessions require signing in again.

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
| POST   | `/auth/register`              | –    | `{email, password}` → `data` account + session cookie|
| POST   | `/auth/login`                 | –    | `{email, password}` → `data` account + session cookie|
| GET    | `/wallets`                    | ✓    | list wallets                          |
| POST   | `/wallets`                    | ✓    | `{name, chain}` (`stellar`)           |
| GET    | `/wallets/:id`                | ✓    | one wallet                            |
| GET    | `/wallets/:id/transactions`   | ✓    | wallet transactions                   |
| POST   | `/transactions`               | ✓    | `{wallet_id, to, amount, memo}`       |
| GET    | `/transactions/:id`           | ✓    | one transaction                       |
| GET    | `/cluster`                    | –    | real MPC peer/node status             |

Outgoing Stellar transactions move through `signing → broadcast → confirmed`
from real MPC result events.
