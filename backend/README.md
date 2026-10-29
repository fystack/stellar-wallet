# Stellar Wallet Backend

Go (Gin) + SQLite API. Keys are generated and transactions signed by an mpcium
cluster (2-of-3) over NATS; updates reach the UI over SSE. Testnet only.

## Run

`./start.sh` at the repo root starts everything (NATS, Consul, MPC nodes, this
backend, the UI). To run only the backend against an existing cluster:

```bash
cp config.example.yaml config.yaml   # set nats_url, consul_addr, initiator_key
go run .
```

All settings come from `config.yaml`; environment variables are not read.

## Verify

```bash
go test ./... && go vet ./... && gofmt -l .
```

## Layout

```text
main.go          load config and infrastructure
internal/
  app/           wire services, handlers and MPC callbacks; start background jobs
  handler/       HTTP: bind input, call one service method, write the response (routes in router.go)
  service/       business logic; no Gin or net/http
  repository/    SQL only (sqlite/), plus the Store contract and sentinel errors
  domain/        models and typed status constants
  apperror/      business errors with stable codes
  httpapi/       JSON envelopes and error-to-HTTP mapping
  auth/          OAuth providers and cookie sessions
  chain/         Stellar Horizon client and transaction builders
  mpc/ cluster/  mpcium adapter and node health
  realtime/      per-user SSE hub
  mail/          SMTP mailer and the HTML/plain-text email template
```

Coding rules for this layout are in [CLAUDE.md](CLAUDE.md).

## How it works

- **Workspaces.** Every user has a Personal workspace and can join team
  workspaces by invite (single-use links, 7 days, emailed when `smtp` is set).
  Creating or joining a workspace never creates a wallet; owners create the
  team's wallets.
  Details: [docs/workspaces.md](../docs/workspaces.md).
- **Roles.** `owner` manages members, wallets and the approval policy;
  `operator` sends payments and approves others'; `member` is read-only.
  Swaps and trustlines are owner-only.
- **MPC queues.** Keygen and signing requests go through `dispatchQueue`
  (`service/dispatch_queue.go`) so a burst never floods mpcium: one keygen at a
  time (`keygen_queue.go`) and two signatures at a time (`sign_queue.go`). The
  next request leaves when one completes, fails or times out.
- **Signing.** Payments, swaps and trustlines build an envelope, then go through
  `TransactionService.dispatch`: persist, MPC sign, broadcast, with timeouts and
  restart recovery. Status moves `signing → broadcast → confirmed` (or `failed`).
- **Approvals.** A team workspace can set a threshold in XLM, the approvals
  needed, and whether smaller XLM payments also need approval; non-XLM payments
  always do. A held payment is saved as a payment request with its inputs only,
  since envelopes expire 300 s after they are built. The last approval claims the
  request in the same SQL transaction as the vote, so it is sent at most once.
  The request stays `executing` until its transaction confirms (`executed`) or fails (`failed`).
  `POST /transactions` returns `201 {transaction}` when sent, `202 {request}` when held.
- **Balances.** A refresher polls Horizon every 30 s, only for wallets in
  workspaces someone has open; public Horizon allows 2,000 requests per 5 minutes per IP.
- **Login.** Password, Google and GitHub, with HttpOnly cookie sessions.
  Setup: [docs/oauth.md](../docs/oauth.md).

## API contract

Routes live in `internal/handler/router.go` under `/api/v1`.

```json
{"data": {"id": "wallet-1"}}
{"data": [], "meta": {"next_cursor": null, "has_more": false}}
{"error": {"code": "wallet_not_found", "message": "Wallet not found."}}
```

- Wallet transactions and payment request history are paged newest first:
  `?limit=` (default 50, max 100) and `?cursor=` taken from `meta.next_cursor`.
- Clients branch on `error.code`, never the message. Unexpected failures return
  `500 internal_error` without internal details.
- Status codes: `201` create, `200` read/action, `204` no body, `400` invalid,
  `401` signed out, `403` forbidden, `404` missing, `409` conflict, `502`
  upstream failure, `503` MPC below quorum.
- SSE events are `{"type": ..., "data": ...}` (`tx`, `wallet`, `payment_request`).
  OAuth start and callback use redirects.
