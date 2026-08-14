package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	const schema = `
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
	return db, nil
}
