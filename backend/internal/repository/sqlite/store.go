package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (s *Store) CustomAssets() ([]domain.CustomAsset, error) {
	rows, err := s.db.Query(`SELECT code, issuer FROM custom_assets ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list custom assets: %w", err)
	}
	defer rows.Close()

	assets := []domain.CustomAsset{}
	for rows.Next() {
		var asset domain.CustomAsset
		if err := rows.Scan(&asset.Code, &asset.Issuer); err != nil {
			return nil, fmt.Errorf("scan custom asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom assets: %w", err)
	}
	return assets, nil
}

func (s *Store) AddAsset(asset domain.CustomAsset) error {
	return exec(s.db, "add asset",
		`INSERT OR IGNORE INTO custom_assets (code, issuer) VALUES (?, ?)`, asset.Code, asset.Issuer,
	)
}

func (s *Store) RemoveAsset(code, issuer string) error {
	return exec(s.db, "remove asset", `DELETE FROM custom_assets WHERE code = ? AND issuer = ?`, code, issuer)
}

const walletColumns = `id, user_id, workspace_id, name, chain, symbol, address, pubkey, balance, status, created_at`

func scanWallet(scanner interface{ Scan(...any) error }) (domain.Wallet, error) {
	var wallet domain.Wallet
	err := scanner.Scan(
		&wallet.ID, &wallet.UserID, &wallet.WorkspaceID, &wallet.Name, &wallet.Chain, &wallet.Symbol,
		&wallet.Address, &wallet.Pubkey, &wallet.Balance, &wallet.Status, &wallet.CreatedAt,
	)
	return wallet, err
}

func (s *Store) ListWallets(workspaceID string) ([]domain.Wallet, error) {
	return s.wallets(`SELECT `+walletColumns+` FROM wallets WHERE workspace_id = ? ORDER BY created_at`, workspaceID)
}

func (s *Store) CreateWallet(wallet domain.Wallet) error {
	_, err := s.db.Exec(
		`INSERT INTO wallets (id, user_id, workspace_id, name, chain, symbol, address, pubkey, balance, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wallet.ID, wallet.UserID, wallet.WorkspaceID, wallet.Name, wallet.Chain, wallet.Symbol,
		wallet.Address, wallet.Pubkey, wallet.Balance, wallet.Status, wallet.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	return nil
}

// WalletFor returns the wallet only when userID is a member of its workspace.
func (s *Store) WalletFor(userID, walletID string) (domain.Wallet, bool) {
	wallet, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletColumns+` FROM wallets
		 WHERE id = ? AND workspace_id IN (SELECT workspace_id FROM workspace_members WHERE user_id = ?)`, walletID, userID,
	))
	return wallet, err == nil
}

func (s *Store) WorkspaceRole(workspaceID, userID string) (domain.WorkspaceRole, error) {
	var role domain.WorkspaceRole
	err := s.db.QueryRow(`SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return role, storecontract.ErrNotFound
	}
	if err != nil {
		return role, fmt.Errorf("read workspace role: %w", err)
	}
	return role, nil
}

func (s *Store) WorkspaceMemberIDs(workspaceID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT user_id FROM workspace_members WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	defer rows.Close()
	userIDs := []string{}
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan workspace member: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace members: %w", err)
	}
	return userIDs, nil
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

// ReadyWalletsForUsers returns ready wallets in any workspace the given users belong to.
func (s *Store) ReadyWalletsForUsers(userIDs []string) ([]domain.Wallet, error) {
	if len(userIDs) == 0 {
		return []domain.Wallet{}, nil
	}
	args := []any{domain.WalletReady}
	for _, userID := range userIDs {
		args = append(args, userID)
	}
	return s.wallets(`SELECT `+walletColumns+` FROM wallets WHERE status = ? AND workspace_id IN (
		SELECT workspace_id FROM workspace_members WHERE user_id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(userIDs)), ",")+`))`, args...)
}

func (s *Store) GeneratingWallets() ([]domain.Wallet, error) {
	return s.wallets(`SELECT `+walletColumns+` FROM wallets WHERE status = ?`, domain.WalletGenerating)
}

func (s *Store) wallets(query string, args ...any) ([]domain.Wallet, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	wallets := []domain.Wallet{}
	for rows.Next() {
		wallet, err := scanWallet(rows)
		if err != nil {
			return nil, fmt.Errorf("scan wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallets: %w", err)
	}
	return wallets, nil
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
func (s *Store) DeleteWallet(walletID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM transactions WHERE wallet_id = ?`, walletID); err != nil {
		return fmt.Errorf("delete wallet transactions: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM wallets WHERE id = ?`, walletID); err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	return nil
}

// transition runs a conditional UPDATE and reports whether it changed a row.
func transition(db *sql.DB, action, query string, args ...any) (bool, error) {
	result, err := db.Exec(query, args...)
	if err != nil {
		return false, fmt.Errorf("%s: %w", action, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("%s: %w", action, err)
	}
	return changed == 1, nil
}

func exec(db *sql.DB, action, query string, args ...any) error {
	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	return nil
}
