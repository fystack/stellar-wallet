package sqlite

import (
	"database/sql"
	"testing"
)

func TestOpenCreatesRequiredTables(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	want := []string{"users", "wallets", "settings", "workspace_assets", "transactions"}
	for _, table := range want {
		var count int
		err := db.QueryRow(
			`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %s count = %d, want 1", table, count)
		}
	}
}

func TestOpenMovesSharedAssetsIntoEachWorkspaceOnce(t *testing.T) {
	path := t.TempDir() + "/wallet.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO workspaces (id, name, created_by, created_at) VALUES ('team', 'Team', 'owner', '2026-10-01T00:00:00Z')`,
		`CREATE TABLE custom_assets (code TEXT NOT NULL, issuer TEXT NOT NULL, PRIMARY KEY (code, issuer))`,
		`INSERT INTO custom_assets (code, issuer) VALUES ('EURC', 'GEURISSUER')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	for run := 0; run < 2; run++ {
		db, err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			var copied int
			if err := db.QueryRow(`SELECT COUNT(1) FROM workspace_assets WHERE workspace_id = 'team' AND code = 'EURC'`).Scan(&copied); err != nil || copied != 1 {
				t.Fatalf("shared asset copied = %d, %v", copied, err)
			}
			if _, err := db.Exec(`DELETE FROM workspace_assets WHERE workspace_id = 'team' AND code = 'EURC'`); err != nil {
				t.Fatal(err)
			}
		}
		db.Close()
	}
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var euro int
	if err := db.QueryRow(`SELECT COUNT(1) FROM workspace_assets WHERE workspace_id = 'team' AND code = 'EURC'`).Scan(&euro); err != nil {
		t.Fatal(err)
	}
	if euro != 0 {
		t.Fatal("removed asset came back after a restart")
	}
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'custom_assets'`).Scan(new(string)); err != sql.ErrNoRows {
		t.Fatalf("shared asset table still exists: %v", err)
	}
}
