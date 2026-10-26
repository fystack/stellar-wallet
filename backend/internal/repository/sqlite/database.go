package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

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
		name       TEXT NOT NULL,
		chain      TEXT NOT NULL,
		symbol     TEXT NOT NULL,
		address    TEXT NOT NULL DEFAULT '',
		pubkey     TEXT NOT NULL DEFAULT '',
		balance    TEXT NOT NULL,
		status     TEXT NOT NULL DEFAULT 'ready',
		created_at TEXT NOT NULL
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
	} {
		_, _ = db.Exec(migration)
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
