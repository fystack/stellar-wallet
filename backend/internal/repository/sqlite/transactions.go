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

func (s *TransactionRepository) IncomingTransactionExists(hash string) (bool, error) {
	return s.exists(`SELECT COUNT(1) FROM transactions WHERE tx_hash = ? AND type = ?`, hash, domain.TransactionIn)
}

func (s *TransactionRepository) TrustlineTransactionExists(walletID, symbol string) (bool, error) {
	return s.exists(
		`SELECT COUNT(1) FROM transactions
		 WHERE wallet_id = ? AND symbol = ? AND memo = ? AND status IN (?, ?, ?)`,
		walletID, symbol, domain.TrustlineMemo,
		domain.TransactionSigning, domain.TransactionBroadcast, domain.TransactionConfirmed,
	)
}

func (s *TransactionRepository) exists(query string, args ...any) (bool, error) {
	var count int
	if err := s.db.QueryRow(query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("count transactions: %w", err)
	}
	return count > 0, nil
}

func (s *TransactionRepository) TransactionsByWallet(walletID string) ([]domain.Transaction, error) {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions
		 WHERE wallet_id = ? ORDER BY created_at DESC`, walletID,
	)
}

// TransactionForUser returns the transaction only when userID is a member of its wallet's workspace.
func (s *TransactionRepository) TransactionForUser(transactionID, userID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions
		 WHERE id = ? AND wallet_id IN (
		   SELECT wallets.id FROM wallets
		   JOIN workspace_members ON workspace_members.workspace_id = wallets.workspace_id
		   WHERE workspace_members.user_id = ?)`,
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

func (s *TransactionRepository) PendingTransactions() ([]domain.Transaction, error) {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions WHERE status = ?`, domain.TransactionSigning,
	)
}

func (s *TransactionRepository) transactions(query string, args ...any) ([]domain.Transaction, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	transactions := []domain.Transaction{}
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return transactions, nil
}

// ClaimSignature stores the signature and moves a signing transaction to broadcast.
// It returns false when the transaction already left signing (timed out, failed or a duplicate result).
func (s *TransactionRepository) ClaimSignature(transactionID, signature string) (bool, error) {
	return transition(s.db, "claim signature",
		`UPDATE transactions SET signature = ?, status = ? WHERE id = ? AND status = ?`,
		signature, domain.TransactionBroadcast, transactionID, domain.TransactionSigning,
	)
}

// FailSigning fails a transaction only if it is still waiting for its signature.
func (s *TransactionRepository) FailSigning(transactionID, reason string) (bool, error) {
	return transition(s.db, "fail signing",
		`UPDATE transactions SET status = ?, error = ? WHERE id = ? AND status = ?`,
		domain.TransactionFailed, reason, transactionID, domain.TransactionSigning,
	)
}

func (s *TransactionRepository) ConfirmTransaction(transactionID, hash string) error {
	return exec(s.db, "confirm transaction",
		`UPDATE transactions SET status = ?, tx_hash = ? WHERE id = ?`, domain.TransactionConfirmed, hash, transactionID,
	)
}

func (s *TransactionRepository) FailTransaction(transactionID, reason string) error {
	return exec(s.db, "fail transaction",
		`UPDATE transactions SET status = ?, error = ? WHERE id = ?`, domain.TransactionFailed, reason, transactionID,
	)
}
