package sqlite

import (
	"database/sql"
	"fmt"

	"stellar-wallet-backend/internal/domain"
)

// TransactionRepository persists sends, swaps, trustlines and incoming payments.
type TransactionRepository struct{ db *sql.DB }

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
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

func (s *TransactionRepository) CreateTransaction(transaction domain.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, recv_amount, recv_symbol, memo, status, signature, envelope_xdr, tx_hash, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		transaction.ID, transaction.WalletID, transaction.UserID, transaction.Type,
		transaction.Counterparty, transaction.Amount, transaction.Symbol,
		transaction.RecvAmount, transaction.RecvSymbol, transaction.Memo,
		transaction.Status, transaction.Signature, transaction.EnvelopeXDR,
		transaction.TxHash, transaction.Error, transaction.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (s *TransactionRepository) IncomingTransactionExists(hash string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions WHERE tx_hash = ? AND type = ?`, hash, domain.TransactionIn,
	).Scan(&count)
	return count > 0
}

func (s *TransactionRepository) TrustlineTransactionExists(walletID, symbol string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions
		 WHERE wallet_id = ? AND symbol = ? AND memo = ? AND status IN (?, ?, ?)`,
		walletID, symbol, domain.TrustlineMemo,
		domain.TransactionSigning, domain.TransactionBroadcast, domain.TransactionConfirmed,
	).Scan(&count)
	return count > 0
}

func (s *TransactionRepository) TransactionsByWallet(userID, walletID string) []domain.Transaction {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions
		 WHERE wallet_id = ? AND user_id = ? ORDER BY created_at DESC`, walletID, userID,
	)
}

func (s *TransactionRepository) TransactionForUser(transactionID, userID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ? AND user_id = ?`,
		transactionID, userID,
	))
	return transaction, err == nil
}

func (s *TransactionRepository) TransactionByID(transactionID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ?`, transactionID,
	))
	return transaction, err == nil
}

func (s *TransactionRepository) transactions(query string, args ...any) []domain.Transaction {
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

func (s *TransactionRepository) PendingTransactions() []domain.Transaction {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions WHERE status = ?`, domain.TransactionSigning,
	)
}

func (s *TransactionRepository) SetTransactionStatus(transactionID string, status domain.TransactionStatus) error {
	return exec(s.db, "set transaction status", `UPDATE transactions SET status = ? WHERE id = ?`, status, transactionID)
}

func (s *TransactionRepository) FailTransaction(transactionID, reason string) error {
	return exec(s.db, "fail transaction",
		`UPDATE transactions SET status = ?, error = ? WHERE id = ?`, domain.TransactionFailed, reason, transactionID,
	)
}

func (s *TransactionRepository) SetTransactionSignature(transactionID, signature string) error {
	return exec(s.db, "set transaction signature",
		`UPDATE transactions SET signature = ? WHERE id = ?`, signature, transactionID,
	)
}

func (s *TransactionRepository) BroadcastTransaction(transactionID, hash string) error {
	return exec(s.db, "broadcast transaction",
		`UPDATE transactions SET status = ?, tx_hash = ? WHERE id = ?`, domain.TransactionBroadcast, hash, transactionID,
	)
}
