package sqlite

import (
	"database/sql"
	"errors"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/store"
)

type Store struct {
	db *sql.DB
}

var _ storecontract.Store = (*Store)(nil)

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateUser(user domain.User) error {
	_, err := s.db.Exec(
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)
	return err
}

func (s *Store) UserCredentials(email string) (string, string, error) {
	var userID, passwordHash string
	err := s.db.QueryRow(
		`SELECT id, password_hash FROM users WHERE email = ?`, email,
	).Scan(&userID, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		err = storecontract.ErrNotFound
	}
	return userID, passwordHash, err
}

func (s *Store) Setting(key string) (string, bool) {
	var value string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value); err != nil || value == "" {
		return "", false
	}
	return value, true
}

func (s *Store) SetSetting(key, value string) {
	_, _ = s.db.Exec(
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

func (s *Store) AddAsset(asset domain.CustomAsset) {
	_, _ = s.db.Exec(
		`INSERT OR IGNORE INTO custom_assets (code, issuer) VALUES (?, ?)`, asset.Code, asset.Issuer,
	)
}

func (s *Store) RemoveAsset(code, issuer string) {
	_, _ = s.db.Exec(`DELETE FROM custom_assets WHERE code = ? AND issuer = ?`, code, issuer)
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
		return nil, errors.New("db error")
	}
	defer rows.Close()

	wallets := []domain.Wallet{}
	for rows.Next() {
		wallet, scanErr := scanWallet(rows)
		if scanErr != nil {
			return nil, errors.New("scan error")
		}
		wallets = append(wallets, wallet)
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
	return err
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
	rows, err := s.db.Query(
		`SELECT ` + walletColumns + ` FROM wallets WHERE status = 'ready'`,
	)
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

func (s *Store) SetWalletStatus(walletID, status string) {
	_, _ = s.db.Exec(`UPDATE wallets SET status = ? WHERE id = ?`, status, walletID)
}

func (s *Store) CompleteWalletKeygen(walletID, address, publicKey string) {
	_, _ = s.db.Exec(
		`UPDATE wallets SET address = ?, pubkey = ?, status = 'ready' WHERE id = ?`,
		address, publicKey, walletID,
	)
}

func (s *Store) SetWalletBalance(walletID, balance string) {
	_, _ = s.db.Exec(`UPDATE wallets SET balance = ? WHERE id = ?`, balance, walletID)
}

func (s *Store) DeleteWallet(userID, walletID string) {
	_, _ = s.db.Exec(`DELETE FROM transactions WHERE wallet_id = ? AND user_id = ?`, walletID, userID)
	_, _ = s.db.Exec(`DELETE FROM wallets WHERE id = ? AND user_id = ?`, walletID, userID)
}

const transactionColumns = `id, wallet_id, user_id, type, counterparty, amount, symbol, COALESCE(recv_amount, ''), COALESCE(recv_symbol, ''), COALESCE(memo, ''), status, signature, envelope_xdr, tx_hash, error, created_at`

func scanTransaction(scanner interface{ Scan(...any) error }) (domain.Transaction, error) {
	var transaction domain.Transaction
	err := scanner.Scan(
		&transaction.ID, &transaction.WalletID, &transaction.UserID, &transaction.Type,
		&transaction.Counterparty, &transaction.Amount, &transaction.Symbol,
		&transaction.RecvAmount, &transaction.RecvSymbol, &transaction.Memo,
		&transaction.Status, &transaction.Signature, &transaction.EnvelopeXDR,
		&transaction.TxHash, &transaction.Error, &transaction.CreatedAt,
	)
	return transaction, err
}

func (s *Store) CreateTransaction(transaction domain.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, recv_amount, recv_symbol, memo, status, signature, envelope_xdr, tx_hash, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		transaction.ID, transaction.WalletID, transaction.UserID, transaction.Type,
		transaction.Counterparty, transaction.Amount, transaction.Symbol,
		transaction.RecvAmount, transaction.RecvSymbol, transaction.Memo,
		transaction.Status, transaction.Signature, transaction.EnvelopeXDR,
		transaction.TxHash, transaction.Error, transaction.CreatedAt,
	)
	return err
}

func (s *Store) IncomingTransactionExists(hash string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions WHERE tx_hash = ? AND type = 'in'`, hash,
	).Scan(&count)
	return count > 0
}

func (s *Store) TrustlineTransactionExists(walletID, symbol string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions
		 WHERE wallet_id = ? AND symbol = ? AND memo = 'Add trustline'
		   AND status IN ('signing','broadcast','confirmed')`, walletID, symbol,
	).Scan(&count)
	return count > 0
}

func (s *Store) TransactionsByWallet(userID, walletID string) []domain.Transaction {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions
		 WHERE wallet_id = ? AND user_id = ? ORDER BY created_at DESC`, walletID, userID,
	)
}

func (s *Store) TransactionForUser(transactionID, userID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ? AND user_id = ?`,
		transactionID, userID,
	))
	return transaction, err == nil
}

func (s *Store) TransactionByID(transactionID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ?`, transactionID,
	))
	return transaction, err == nil
}

func (s *Store) transactions(query string, args ...any) []domain.Transaction {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return []domain.Transaction{}
	}
	defer rows.Close()

	transactions := []domain.Transaction{}
	for rows.Next() {
		if transaction, scanErr := scanTransaction(rows); scanErr == nil {
			transactions = append(transactions, transaction)
		}
	}
	return transactions
}

func (s *Store) SetTransactionStatus(transactionID, status string) {
	_, _ = s.db.Exec(`UPDATE transactions SET status = ? WHERE id = ?`, status, transactionID)
}

func (s *Store) FailTransaction(transactionID, reason string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET status = 'failed', error = ? WHERE id = ?`, reason, transactionID,
	)
}

func (s *Store) SetTransactionSignature(transactionID, signature string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET signature = ? WHERE id = ?`, signature, transactionID,
	)
}

func (s *Store) BroadcastTransaction(transactionID, hash string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET status = 'broadcast', tx_hash = ? WHERE id = ?`, hash, transactionID,
	)
}
