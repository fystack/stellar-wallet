package sqlite

import (
	"database/sql"
	"stellar-wallet-backend/internal/domain"
)

// PaymentRepository persists transactions, including sends, swaps and trustlines.
type PaymentRepository struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
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

func (s *PaymentRepository) CreateTransaction(transaction domain.Transaction) error {
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

func (s *PaymentRepository) IncomingTransactionExists(hash string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions WHERE tx_hash = ? AND type = 'in'`, hash,
	).Scan(&count)
	return count > 0
}

func (s *PaymentRepository) TrustlineTransactionExists(walletID, symbol string) bool {
	var count int
	_ = s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions
		 WHERE wallet_id = ? AND symbol = ? AND memo = 'Add trustline'
		   AND status IN ('signing','broadcast','confirmed')`, walletID, symbol,
	).Scan(&count)
	return count > 0
}

func (s *PaymentRepository) TransactionsByWallet(userID, walletID string) []domain.Transaction {
	return s.transactions(
		`SELECT `+transactionColumns+` FROM transactions
		 WHERE wallet_id = ? AND user_id = ? ORDER BY created_at DESC`, walletID, userID,
	)
}

func (s *PaymentRepository) TransactionForUser(transactionID, userID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ? AND user_id = ?`,
		transactionID, userID,
	))
	return transaction, err == nil
}

func (s *PaymentRepository) TransactionByID(transactionID string) (domain.Transaction, bool) {
	transaction, err := scanTransaction(s.db.QueryRow(
		`SELECT `+transactionColumns+` FROM transactions WHERE id = ?`, transactionID,
	))
	return transaction, err == nil
}

func (s *PaymentRepository) transactions(query string, args ...any) []domain.Transaction {
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

func (s *PaymentRepository) PendingTransactions() []domain.Transaction {
	return s.transactions(
		`SELECT ` + transactionColumns + ` FROM transactions WHERE status = 'signing'`,
	)
}

func (s *PaymentRepository) SetTransactionStatus(transactionID, status string) {
	_, _ = s.db.Exec(`UPDATE transactions SET status = ? WHERE id = ?`, status, transactionID)
}

func (s *PaymentRepository) FailTransaction(transactionID, reason string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET status = 'failed', error = ? WHERE id = ?`, reason, transactionID,
	)
}

func (s *PaymentRepository) SetTransactionSignature(transactionID, signature string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET signature = ? WHERE id = ?`, signature, transactionID,
	)
}

func (s *PaymentRepository) BroadcastTransaction(transactionID, hash string) {
	_, _ = s.db.Exec(
		`UPDATE transactions SET status = 'broadcast', tx_hash = ? WHERE id = ?`, hash, transactionID,
	)
}
