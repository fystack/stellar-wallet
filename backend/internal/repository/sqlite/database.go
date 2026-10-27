package sqlite

import (
	"database/sql"
	"fmt"

	"stellar-wallet-backend/internal/domain"

	_ "modernc.org/sqlite"
)

var personalPrefix = domain.PersonalWorkspaceID("")

const personalName = "Personal"

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	const schema = `
 CREATE TABLE IF NOT EXISTS auth_identities (
  provider TEXT NOT NULL,
  subject TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id),
  email TEXT NOT NULL,
  PRIMARY KEY(provider,subject),
  UNIQUE(user_id,provider)
 );
 CREATE TABLE IF NOT EXISTS auth_sessions (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
 );
 CREATE INDEX IF NOT EXISTS auth_sessions_expiry ON auth_sessions(expires_at);
 CREATE TABLE IF NOT EXISTS oauth_flows (
  state_hash TEXT PRIMARY KEY,
  browser_hash TEXT NOT NULL,
  provider TEXT NOT NULL,
  verifier TEXT NOT NULL,
  nonce TEXT NOT NULL,
  session_hash TEXT NOT NULL DEFAULT '',
  user_id TEXT NOT NULL DEFAULT '',
  expires_at INTEGER NOT NULL
 );
 CREATE INDEX IF NOT EXISTS oauth_flows_expiry ON oauth_flows(expires_at);
	CREATE TABLE IF NOT EXISTS users (
		id            TEXT PRIMARY KEY,
		email         TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at    TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS wallets (
		id         TEXT PRIMARY KEY,
		user_id    TEXT NOT NULL,
		workspace_id TEXT NOT NULL DEFAULT '',
		name       TEXT NOT NULL,
		chain      TEXT NOT NULL,
		symbol     TEXT NOT NULL,
		address    TEXT NOT NULL DEFAULT '',
		pubkey     TEXT NOT NULL DEFAULT '',
		balance    TEXT NOT NULL,
		status     TEXT NOT NULL DEFAULT 'ready',
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS workspaces (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS workspace_members (
		workspace_id TEXT NOT NULL REFERENCES workspaces(id),
		user_id      TEXT NOT NULL,
		role         TEXT NOT NULL,
		joined_at    TEXT NOT NULL,
		PRIMARY KEY (workspace_id, user_id)
	);
	CREATE INDEX IF NOT EXISTS workspace_members_user ON workspace_members(user_id);
	CREATE TABLE IF NOT EXISTS workspace_invites (
		id           TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL REFERENCES workspaces(id),
		email        TEXT NOT NULL,
		token_hash   TEXT UNIQUE NOT NULL,
		invited_by   TEXT NOT NULL,
		created_at   TEXT NOT NULL,
		expires_at   INTEGER NOT NULL,
		accepted_at  TEXT NOT NULL DEFAULT '',
		revoked_at   TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS workspace_invites_workspace ON workspace_invites(workspace_id);
	CREATE TABLE IF NOT EXISTS workspace_approval_policies (
		workspace_id       TEXT PRIMARY KEY REFERENCES workspaces(id),
		threshold_xlm      TEXT NOT NULL,
		required_approvals INTEGER NOT NULL,
		below_threshold    TEXT NOT NULL,
		updated_by         TEXT NOT NULL,
		updated_at         TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS payment_requests (
		id                 TEXT PRIMARY KEY,
		workspace_id       TEXT NOT NULL REFERENCES workspaces(id),
		wallet_id          TEXT NOT NULL,
		initiator_id       TEXT NOT NULL,
		destination        TEXT NOT NULL,
		amount             TEXT NOT NULL,
		asset_code         TEXT NOT NULL DEFAULT '',
		asset_issuer       TEXT NOT NULL DEFAULT '',
		symbol             TEXT NOT NULL,
		memo               TEXT NOT NULL DEFAULT '',
		memo_type          TEXT NOT NULL DEFAULT '',
		required_approvals INTEGER NOT NULL,
		status             TEXT NOT NULL,
		transaction_id     TEXT NOT NULL DEFAULT '',
		reason             TEXT NOT NULL DEFAULT '',
		created_at         TEXT NOT NULL,
		expires_at         INTEGER NOT NULL,
		decided_at         TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS payment_requests_workspace ON payment_requests(workspace_id, status);
	CREATE TABLE IF NOT EXISTS payment_request_votes (
		request_id TEXT NOT NULL REFERENCES payment_requests(id),
		user_id    TEXT NOT NULL,
		decision   TEXT NOT NULL,
		created_at TEXT NOT NULL,
		PRIMARY KEY (request_id, user_id)
	);
	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS custom_assets (
		code   TEXT NOT NULL,
		issuer TEXT NOT NULL,
		PRIMARY KEY (code, issuer)
	);
	CREATE TABLE IF NOT EXISTS transactions (
		id           TEXT PRIMARY KEY,
		wallet_id    TEXT NOT NULL,
		user_id      TEXT NOT NULL,
		type         TEXT NOT NULL,
		counterparty TEXT NOT NULL,
		amount       TEXT NOT NULL,
		symbol       TEXT NOT NULL,
		recv_amount  TEXT NOT NULL DEFAULT '',
		recv_symbol  TEXT NOT NULL DEFAULT '',
		memo         TEXT,
		status       TEXT NOT NULL,
		signature    TEXT NOT NULL DEFAULT '',
		envelope_xdr TEXT NOT NULL DEFAULT '',
		tx_hash      TEXT NOT NULL DEFAULT '',
		error        TEXT NOT NULL DEFAULT '',
		created_at   TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	// Idempotent migrations for databases created before these columns existed.
	// SQLite has no "ADD COLUMN IF NOT EXISTS"; a duplicate-column error is fine.
	for _, migration := range []string{
		`ALTER TABLE transactions ADD COLUMN recv_amount TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE transactions ADD COLUMN recv_symbol TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE wallets ADD COLUMN workspace_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE auth_sessions ADD COLUMN workspace_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE workspace_invites ADD COLUMN role TEXT NOT NULL DEFAULT 'member'`,
	} {
		_, _ = db.Exec(migration)
	}
	if err := backfillPersonalWorkspaces(db); err != nil {
		db.Close()
		return nil, err
	}
	// Seed default Stellar testnet assets so they're swappable/sendable out of the
	// box. INSERT OR IGNORE keeps it idempotent; users can still delete them.
	const testnetMarketMaker = "GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER"
	for _, seed := range [][2]string{
		{"USDC", "GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5"},
		{"USDT", testnetMarketMaker},
		{"BTC", testnetMarketMaker},
		{"ETH", testnetMarketMaker},
	} {
		_, _ = db.Exec(`INSERT OR IGNORE INTO custom_assets (code, issuer) VALUES (?, ?)`, seed[0], seed[1])
	}
	return db, nil
}

// backfillPersonalWorkspaces is idempotent: it runs on every start for pre-workspace databases.
func backfillPersonalWorkspaces(db *sql.DB) error {
	statements := []struct {
		action string
		query  string
		args   []any
	}{
		{"create personal workspaces", `INSERT OR IGNORE INTO workspaces (id, name, created_by, created_at)
			SELECT ? || id, ?, id, created_at FROM users`, []any{personalPrefix, personalName}},
		{"add personal workspace owners", `INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, joined_at)
			SELECT ? || id, id, ?, created_at FROM users`, []any{personalPrefix, domain.RoleOwner}},
		{"index wallet workspaces", `CREATE INDEX IF NOT EXISTS wallets_workspace ON wallets(workspace_id)`, nil},
		{"move wallets into personal workspaces", `UPDATE wallets SET workspace_id = ? || user_id WHERE workspace_id = ''`, []any{personalPrefix}},
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement.query, statement.args...); err != nil {
			return fmt.Errorf("%s: %w", statement.action, err)
		}
	}
	return nil
}
