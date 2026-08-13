# Stellar Wallet Backend

Go (Gin) + SQLite. JWT auth. Simulates the MPC signing lifecycle.

## Run

```bash
# Port 8080 is taken by mpcium's `api` on this machine, so use 8090.
ADDR=:8090 go run .
```

Env vars: `ADDR` (default `:8080`), `DB_PATH` (default `wallet.db`),
`CORS_ORIGIN` (default `http://localhost:5173`).

## Endpoints (`/api/v1`)

| Method | Path                          | Auth | Body / notes                          |
| ------ | ----------------------------- | ---- | ------------------------------------- |
| POST   | `/auth/register`              | –    | `{email, password}` → `{access_token}`|
| POST   | `/auth/login`                 | –    | `{email, password}` → `{access_token}`|
| GET    | `/wallets`                    | ✓    | list wallets                          |
| POST   | `/wallets`                    | ✓    | `{name, chain}` (stellar/ethereum/solana/bitcoin) |
| GET    | `/wallets/:id`                | ✓    | one wallet                            |
| GET    | `/wallets/:id/transactions`   | ✓    | wallet transactions                   |
| POST   | `/transactions`               | ✓    | `{wallet_id, to, amount, memo}`       |
| GET    | `/transactions/:id`           | ✓    | one transaction                       |
| GET    | `/cluster`                    | –    | MPC node status                       |

Outgoing transactions advance `policy_check → signing → broadcast → confirmed`
(~1.4s per step) in a background goroutine — poll the transaction to watch it.
