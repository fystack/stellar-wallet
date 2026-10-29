# Social login and session contract

D1 uses Google and GitHub authorization-code flows with PKCE. Google ID tokens
are verified for signature, issuer, audience, expiration and nonce. GitHub user
IDs and verified email addresses are read from its authenticated API.

Accounts retain local user IDs. Provider identities are keyed by (provider,
subject), never by email. An existing email requires explicit linking from an
existing, recently authenticated session. Linking does not create another wallet.

Session tokens are opaque HttpOnly SameSite=Lax cookies; only token hashes are
stored in SQLite. Logout revokes the session. OAuth state expires after ten
minutes, is bound to a browser cookie and is consumed once. Link callbacks also
require the same still-active session that started the flow.

All unsafe API requests require X-CSRF-Protection: 1 and, when present, an allowed
Origin. Only explicitly configured origins receive credentialed CORS responses.
Provider tokens stay in the backend and are not persisted. Old JWTs are retired;
existing users sign in again after deployment. No MPC wallet/share data changes.

References:
- https://developers.google.com/identity/openid-connect/openid-connect
- https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
- https://docs.github.com/en/rest/users/emails#list-email-addresses-for-the-authenticated-user
- https://pkg.go.dev/golang.org/x/oauth2
- https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc

## Local setup

1. Copy `backend/config.example.yaml` to the ignored `backend/config.yaml`.
2. In Google Cloud Console create an OAuth client of type **Web application**;
   request only `openid email profile`. Configure the consent screen/test users.
   Add this exact authorized redirect URI for the default local setup:
   `http://localhost:8090/api/v1/auth/google/callback`.
3. In GitHub Developer settings create an **OAuth App**. Set its callback to
   `http://localhost:8090/api/v1/auth/github/callback`.
   The application requests only `read:user user:email` (no repository scopes).
4. Put both client IDs/secrets into your private backend config:

```yaml
public_url: "http://localhost:8090"
frontend_url: "http://localhost:5173"
cors_origin: "http://localhost:5173"
cookie_secure: false

oauth:
  google:
    client_id: "YOUR_GOOGLE_CLIENT_ID"
    client_secret: "YOUR_GOOGLE_CLIENT_SECRET"
  github:
    client_id: "YOUR_GITHUB_CLIENT_ID"
    client_secret: "YOUR_GITHUB_CLIENT_SECRET"
```

5. Start the backend against your MPC cluster, then run `npm run dev` in `ui/`.
   Open `http://localhost:5173`. Keep the same hostname for frontend and backend
   (`localhost` and `127.0.0.1` are different cookie hosts).
6. A provider with empty credentials is disabled and its button is unavailable.
   A partially configured provider fails startup rather than silently misbehaving.

For Docker's nginx proxy use `http://localhost:8080` for **all three URL fields**
and register the callbacks on port 8080. Use an ignored/private configuration
file and mount it to `/app/config.yaml` instead of committing client secrets into
`config.docker.yaml`. The UI build uses a same-origin API base.

For a public deployment use HTTPS, `cookie_secure: true`, the public callback
URLs, and an exact frontend CORS origin. Serve frontend/API on the same origin
(recommended) or same site because sessions use SameSite=Lax cookies. Wildcard
CORS origins and public plain-HTTP origins are rejected by configuration checks.

## Reproduction and acceptance checks

- Register with email/password, create a wallet with the existing flow.
- Settings → Linked accounts → Connect Google → authorize. The same local user ID
  and wallet remain. Repeat with GitHub (its email can differ when linking).
- Sign out, then sign in through each linked provider. The same wallet is listed.
- For a new provider identity with a new verified email, a new account is created;
  creating its first wallet remains an explicit action.
- A social login matching an existing email without a linked identity shows
  `account_link_required`; sign in using the old method, then link from Settings.
- After ten minutes, linking requires fresh authentication. Password accounts can
  confirm their password in Settings; social-only accounts sign out/in again.
- Denying consent shows a friendly error. Expired, replayed, different-provider or
  wrong-browser states are rejected. Logging out during linking invalidates it.
- Logout revokes only the current session; old cookies fail `/auth/me` and wallet
  APIs immediately. SSE validates the session before each event and on heartbeat.
- Passkeys, unlinking providers and password reset are outside this change.

## API

Normal JSON endpoints retain the `{data}` / `{error:{code,message}}` contract.
Register/login return account details and set a cookie; they no longer return
`access_token`. No browser token goes in a URL or localStorage.

| Method | Route | Behavior |
|---|---|---|
| POST | `/auth/register` | Create account + session (`201`) |
| POST | `/auth/login` | Password login + session (`200`) |
| GET | `/auth/me` | Current account and linked providers |
| POST | `/auth/logout` | Revoke current session, clear cookie (`204`) |
| GET | `/auth/providers` | Enabled/disabled providers |
| GET | `/auth/:provider/start` | Start sign-in (`302`) |
| GET | `/auth/:provider/callback` | Verify and redirect to frontend (`302`) |
| POST | `/auth/:provider/link` | Require recent session, return authorization URL |
| POST | `/auth/reauthenticate` | Confirm password, rotate session (`204`) |

All routes use `/api/v1`. Fetches use `credentials: 'include'`. Unsafe requests
include `X-CSRF-Protection: 1`; wrong/missing headers or disallowed Origins return
`403 csrf_failed`. Authentication endpoints are limited to 30 attempts/minute per
direct peer in memory (`429 rate_limited`); a reverse proxy shares that peer limit.
The limiter resets on process restart. OAuth callbacks do not consume this limit.

## Migration and verification

SQLite setup adds `auth_identities`, `auth_sessions` and `oauth_flows` plus expiry
indexes idempotently. It does not rewrite existing users, password hashes, wallet
ownership, transactions, MPC keys or shares. Existing password users need no data
backfill. Old JWT sessions are intentionally retired; users sign in again. Remove
obsolete `auth_secret` from config and replace any `cors_origin: "*"` with the
exact browser origin. Keep a normal database backup before upgrading.

```bash
cd backend
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go build ./...
cd ../ui
npm test
npm run build
```

Tests use fake OAuth providers and signed local OIDC tokens; live Google/GitHub
verification additionally requires real configured client credentials and consent.
The GitHub adapter pins REST API version `2026-03-10`.
