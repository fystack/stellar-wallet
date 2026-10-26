package sqlite

import (
	"database/sql"
	"fmt"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

type Store struct {
	db *sql.DB
}

var _ storecontract.Store = (*Store)(nil)

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Setting(key string) (string, bool) {
	var value string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value); err != nil || value == "" {
		return "", false
	}
	return value, true
}

func (s *Store) SetSetting(key, value string) error {
	return exec(s.db, "set setting",
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value,
	)
}

func (s *Store) CustomAssets() []domain.CustomAsset {
	rows, err := s.db.Query(`SELECT code, issuer FROM custom_assets ORDER BY code`)
	if err != nil {
		return []domain.CustomAsset{}
	}
	defer rows.Close()

	assets := []domain.CustomAsset{}
	for rows.Next() {
		var asset domain.CustomAsset
		if rows.Scan(&asset.Code, &asset.Issuer) == nil {
			assets = append(assets, asset)
		}
	}
	return assets
}

func (s *Store) AddAsset(asset domain.CustomAsset) error {
	return exec(s.db, "add asset",
		`INSERT OR IGNORE INTO custom_assets (code, issuer) VALUES (?, ?)`, asset.Code, asset.Issuer,
	)
}

func (s *Store) RemoveAsset(code, issuer string) error {
	return exec(s.db, "remove asset", `DELETE FROM custom_assets WHERE code = ? AND issuer = ?`, code, issuer)
}

const walletColumns = `id, user_id, name, chain, symbol, address, pubkey, balance, status, created_at`

func scanWallet(scanner interface{ Scan(...any) error }) (domain.Wallet, error) {
	var wallet domain.Wallet
	err := scanner.Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Chain, &wallet.Symbol,
		&wallet.Address, &wallet.Pubkey, &wallet.Balance, &wallet.Status, &wallet.CreatedAt,
	)
	return wallet, err
}

func (s *Store) ListWallets(userID string) ([]domain.Wallet, error) {
	rows, err := s.db.Query(
		`SELECT `+walletColumns+` FROM wallets WHERE user_id = ? ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	wallets := []domain.Wallet{}
	for rows.Next() {
		wallet, scanErr := scanWallet(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan wallet: %w", scanErr)
		}
		wallets = append(wallets, wallet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallets: %w", err)
	}
	return wallets, nil
}

func (s *Store) CreateWallet(wallet domain.Wallet) error {
	_, err := s.db.Exec(
		`INSERT INTO wallets (id, user_id, name, chain, symbol, address, pubkey, balance, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wallet.ID, wallet.UserID, wallet.Name, wallet.Chain, wallet.Symbol,
		wallet.Address, wallet.Pubkey, wallet.Balance, wallet.Status, wallet.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	return nil
}

func (s *Store) WalletFor(userID, walletID string) (domain.Wallet, bool) {
	wallet, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletColumns+` FROM wallets WHERE id = ? AND user_id = ?`, walletID, userID,
	))
	return wallet, err == nil
}

func (s *Store) WalletByID(walletID string) (domain.Wallet, bool) {
	wallet, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletColumns+` FROM wallets WHERE id = ?`, walletID,
	))
	return wallet, err == nil
}

func (s *Store) WalletByAddress(address string) (domain.Wallet, bool) {
	wallet, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletColumns+` FROM wallets WHERE address = ?`, address,
	))
	return wallet, err == nil
}

func (s *Store) ReadyWallets() []domain.Wallet {
	return s.walletsWithStatus(domain.WalletReady)
}

func (s *Store) GeneratingWallets() []domain.Wallet {
	return s.walletsWithStatus(domain.WalletGenerating)
}

func (s *Store) walletsWithStatus(status domain.WalletStatus) []domain.Wallet {
	rows, err := s.db.Query(`SELECT `+walletColumns+` FROM wallets WHERE status = ?`, status)
	if err != nil {
		return nil
	}
	defer rows.Close()

	wallets := []domain.Wallet{}
	for rows.Next() {
		if wallet, scanErr := scanWallet(rows); scanErr == nil {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}

func (s *Store) SetWalletStatus(walletID string, status domain.WalletStatus) error {
	return exec(s.db, "set wallet status", `UPDATE wallets SET status = ? WHERE id = ?`, status, walletID)
}

func (s *Store) CompleteWalletKeygen(walletID, address, publicKey string) error {
	return exec(s.db, "complete wallet keygen",
		`UPDATE wallets SET address = ?, pubkey = ?, status = ? WHERE id = ?`,
		address, publicKey, domain.WalletReady, walletID,
	)
}

func (s *Store) SetWalletBalance(walletID, balance string) error {
	return exec(s.db, "set wallet balance", `UPDATE wallets SET balance = ? WHERE id = ?`, balance, walletID)
}

// DeleteWallet removes the wallet and its transaction history atomically.
func (s *Store) DeleteWallet(userID, walletID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM transactions WHERE wallet_id = ? AND user_id = ?`, walletID, userID); err != nil {
		return fmt.Errorf("delete wallet transactions: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM wallets WHERE id = ? AND user_id = ?`, walletID, userID); err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	return nil
}

func exec(db *sql.DB, action, query string, args ...any) error {
	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	return nil
}
