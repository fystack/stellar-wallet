# Backend rules

Go (Gin) + SQLite API. Testnet only. Follow these when adding or changing backend code.

## Layers

- `handler/` → `service/` → `repository/`. Never skip a layer.
- One handler struct per API area (`WalletHandler`, `TransactionHandler`, ...), holding only the services it calls. Every route is registered in `handler/router.go`; nowhere else.
- A handler method does exactly three things: bind input, call one service method, write the response. No validation, no store/chain/MPC access, no business branching.
- Services hold the business logic and never import Gin or `net/http`.
- Repositories only run SQL. No business rules, no publishing events.
- Wire new services in `app/server.go`; `main.go` only loads config and infrastructure.

## Errors and responses

- Business failures are `apperror` values defined in `apperror/catalog.go` (`apperror.X.New()` or `.Wrap(cause)`). Reuse an existing code before adding one; codes are a client contract, never rename them.
- Handlers send errors only through `httpapi.Error(c, err)` and success only through `httpapi.Success(c, status, data)`. Never call `c.JSON` directly.
- Wrap technical causes with `%w` and context: `fmt.Errorf("persist wallet: %w", err)`.
- Every repository write returns `error`. Never discard it with `_ =`:
  - request path: return it to the handler;
  - background path (MPC callbacks, timeouts, balance refresh): `logError("<action> id=<id>", err)`.
- Do not turn an upstream failure into a fake value (for example a `"0.00"` balance with a nil error). Return the error.

## Domain

- Use the typed constants in `domain/status.go` (`domain.WalletReady`, `domain.TransactionSigning`, `domain.TransactionSwap`, ...). Never write status/type strings by hand, in Go or in SQL; pass them as query parameters.
- Check wallet access with `ownedWallet` / `readyWallet` (`service/ownership.go`); every wallet-scoped query is also filtered by `user_id`.
- Check MPC quorum with `requireQuorum` before creating any record that needs signing or keygen.
- Anything that needs signing builds its envelope, then goes through `TransactionService.dispatch`. Do not re-implement persist/sign/timeout/publish.
- Amounts are decimal strings. Never round a user amount through `float64` before building a transaction; Stellar supports 7 decimal places.

## External boundaries and tests

- Services depend on small interfaces for Stellar, the signer and the cluster (`PaymentChain`, `SwapChain`, `TrustlineChain`, `PaymentSigner`, `SigningCluster`). Add a method to the narrowest interface that needs it.
- Every new service method gets a table-driven test using stubs and a temp SQLite DB (`sqlite.Open(t.TempDir() + "/wallet.db")`). Cover the success path and each `apperror` code it can return, and assert nothing was dispatched on rejection.
- HTTP contract changes get a handler test through `NewRouter(...)`.
- Before finishing: `gofmt -l .`, `go vet ./...`, `go test ./...` must be clean.

## Security

- Never log OAuth codes, state, tokens, cookies, signatures or raw provider bodies.
- Unsafe methods stay behind the CSRF middleware; do not add routes that bypass it.
- Secrets live only in the git-ignored `config.yaml`. Config comes from `config.yaml` only, not environment variables.

## Style

- Names are full words matching the domain (`transaction`, not `tx`; `wallet`, not `w`), except receivers (`s` for services, `h` for handlers).
- Import groups: stdlib, then `stellar-wallet-backend/...`, then third-party.
- Keep `backend/README.md` in sync when you add a service, handler or convention.
