package sqlite

import (
	"testing"
)

func TestOpenCreatesRequiredTables(t *testing.T) {
	db, err := Open(t.TempDir() + "/wallet.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	want := []string{"users", "wallets", "settings", "custom_assets", "transactions"}
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
