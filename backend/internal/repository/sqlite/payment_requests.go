package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

type PaymentRequestRepository struct{ db *sql.DB }

func NewPaymentRequestRepository(db *sql.DB) *PaymentRequestRepository {
	return &PaymentRequestRepository{db: db}
}

func (s *PaymentRequestRepository) CreatePaymentRequest(request domain.PaymentRequest) error {
	_, err := s.db.Exec(
		`INSERT INTO payment_requests (id, workspace_id, wallet_id, initiator_id, destination, amount, asset_code, asset_issuer,
		 symbol, memo, memo_type, required_approvals, status, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		request.ID, request.WorkspaceID, request.WalletID, request.InitiatorID, request.Destination, request.Amount,
		request.AssetCode, request.AssetIssuer, request.Symbol, request.Memo, request.MemoType, request.RequiredApprovals,
		request.Status, request.CreatedAt, request.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create payment request: %w", err)
	}
	return nil
}

const paymentRequestColumns = `payment_requests.id, payment_requests.workspace_id, payment_requests.wallet_id, COALESCE(wallets.name, ''),
	payment_requests.initiator_id, COALESCE(users.email, ''), payment_requests.destination, payment_requests.amount,
	payment_requests.asset_code, payment_requests.asset_issuer, payment_requests.symbol, payment_requests.memo,
	payment_requests.memo_type, payment_requests.required_approvals, payment_requests.status, payment_requests.transaction_id,
	COALESCE(transactions.status, ''), COALESCE(transactions.tx_hash, ''), payment_requests.reason, payment_requests.created_at,
	payment_requests.expires_at, payment_requests.decided_at`

const paymentRequestFrom = ` FROM payment_requests
	LEFT JOIN wallets ON wallets.id = payment_requests.wallet_id
	LEFT JOIN users ON users.id = payment_requests.initiator_id
	LEFT JOIN transactions ON transactions.id = payment_requests.transaction_id `

func scanPaymentRequest(scanner interface{ Scan(...any) error }) (domain.PaymentRequest, error) {
	var request domain.PaymentRequest
	err := scanner.Scan(&request.ID, &request.WorkspaceID, &request.WalletID, &request.WalletName, &request.InitiatorID,
		&request.InitiatorEmail, &request.Destination, &request.Amount, &request.AssetCode, &request.AssetIssuer, &request.Symbol,
		&request.Memo, &request.MemoType, &request.RequiredApprovals, &request.Status, &request.TransactionID,
		&request.TransactionStatus, &request.TxHash, &request.Reason, &request.CreatedAt, &request.ExpiresAt, &request.DecidedAt)
	return request, err
}

func (s *PaymentRequestRepository) PaymentRequestByID(requestID string) (domain.PaymentRequest, error) {
	request, err := scanPaymentRequest(s.db.QueryRow(`SELECT `+paymentRequestColumns+paymentRequestFrom+`WHERE payment_requests.id = ?`, requestID))
	if errors.Is(err, sql.ErrNoRows) {
		return request, storecontract.ErrNotFound
	}
	if err != nil {
		return request, fmt.Errorf("read payment request: %w", err)
	}
	requests := []domain.PaymentRequest{request}
	if err := s.attachVotes(requests); err != nil {
		return request, err
	}
	return requests[0], nil
}

// OpenPaymentRequests lists pending requests that have not expired.
func (s *PaymentRequestRepository) OpenPaymentRequests(workspaceID string, now int64) ([]domain.PaymentRequest, error) {
	return s.paymentRequests(`WHERE payment_requests.workspace_id = ? AND payment_requests.status = ? AND payment_requests.expires_at > ?
		ORDER BY payment_requests.created_at DESC`, workspaceID, domain.PaymentRequestPending, now)
}

// ClosedPaymentRequests lists decided and expired requests, newest first.
func (s *PaymentRequestRepository) ClosedPaymentRequests(workspaceID string, now int64, before storecontract.Cursor, limit int) ([]domain.PaymentRequest, error) {
	closed := `WHERE payment_requests.workspace_id = ? AND (payment_requests.status != ? OR payment_requests.expires_at <= ?) `
	order := `ORDER BY payment_requests.created_at DESC, payment_requests.id DESC LIMIT ?`
	if before.ID == "" {
		return s.paymentRequests(closed+order, workspaceID, domain.PaymentRequestPending, now, limit)
	}
	return s.paymentRequests(closed+`AND (payment_requests.created_at, payment_requests.id) < (?, ?) `+order,
		workspaceID, domain.PaymentRequestPending, now, before.CreatedAt, before.ID, limit)
}

func (s *PaymentRequestRepository) paymentRequests(where string, args ...any) ([]domain.PaymentRequest, error) {
	rows, err := s.db.Query(`SELECT `+paymentRequestColumns+paymentRequestFrom+where, args...)
	if err != nil {
		return nil, fmt.Errorf("list payment requests: %w", err)
	}
	defer rows.Close()
	requests := []domain.PaymentRequest{}
	for rows.Next() {
		request, err := scanPaymentRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment request: %w", err)
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment requests: %w", err)
	}
	if err := s.attachVotes(requests); err != nil {
		return nil, err
	}
	return requests, nil
}

func (s *PaymentRequestRepository) attachVotes(requests []domain.PaymentRequest) error {
	if len(requests) == 0 {
		return nil
	}
	args := make([]any, len(requests))
	for index, request := range requests {
		args[index] = request.ID
	}
	args = append([]any{domain.RoleOwner, domain.RoleOperator}, args...)
	rows, err := s.db.Query(
		`SELECT payment_request_votes.request_id, payment_request_votes.user_id, COALESCE(users.email, ''), payment_request_votes.decision,
			payment_request_votes.created_at, COALESCE(workspace_members.role IN (?, ?), 0)
		 FROM payment_request_votes
		 JOIN payment_requests ON payment_requests.id = payment_request_votes.request_id
		 LEFT JOIN users ON users.id = payment_request_votes.user_id
		 LEFT JOIN workspace_members ON workspace_members.user_id = payment_request_votes.user_id
			AND workspace_members.workspace_id = payment_requests.workspace_id
		 WHERE payment_request_votes.request_id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(requests)), ",")+`)
		 ORDER BY payment_request_votes.created_at`, args...,
	)
	if err != nil {
		return fmt.Errorf("list votes: %w", err)
	}
	defer rows.Close()
	votes := map[string][]domain.PaymentRequestVote{}
	for rows.Next() {
		var requestID string
		var vote domain.PaymentRequestVote
		if err := rows.Scan(&requestID, &vote.UserID, &vote.Email, &vote.Decision, &vote.CreatedAt, &vote.Counted); err != nil {
			return fmt.Errorf("scan vote: %w", err)
		}
		votes[requestID] = append(votes[requestID], vote)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate votes: %w", err)
	}
	for index := range requests {
		requests[index].Votes = votes[requests[index].ID]
		if requests[index].Votes == nil {
			requests[index].Votes = []domain.PaymentRequestVote{}
		}
	}
	return nil
}

// RecordApproval returns claimed=true only to the vote completing the approvals of current owners and operators.
func (s *PaymentRequestRepository) RecordApproval(requestID, userID, createdAt string, now int64) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("record approval: %w", err)
	}
	defer tx.Rollback()
	if err := insertVote(tx, requestID, userID, domain.VoteApprove, createdAt, now); err != nil {
		return false, err
	}
	result, err := tx.Exec(
		`UPDATE payment_requests SET status = ?, decided_at = ?
		 WHERE id = ? AND status = ? AND required_approvals <= (
			SELECT COUNT(1) FROM payment_request_votes
			JOIN workspace_members ON workspace_members.user_id = payment_request_votes.user_id
				AND workspace_members.workspace_id = payment_requests.workspace_id
			WHERE payment_request_votes.request_id = ? AND payment_request_votes.decision = ? AND workspace_members.role IN (?, ?))`,
		domain.PaymentRequestExecuting, createdAt, requestID, domain.PaymentRequestPending, requestID, domain.VoteApprove, domain.RoleOwner, domain.RoleOperator,
	)
	if err != nil {
		return false, fmt.Errorf("claim payment request: %w", err)
	}
	claimed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claim payment request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("record approval: %w", err)
	}
	return claimed == 1, nil
}

// RecordRejection returns ErrConflict when the request is closed and ErrDuplicate when the user already voted.
func (s *PaymentRequestRepository) RecordRejection(requestID, userID, createdAt string, now int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("record rejection: %w", err)
	}
	defer tx.Rollback()
	if err := insertVote(tx, requestID, userID, domain.VoteReject, createdAt, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE payment_requests SET status = ?, decided_at = ? WHERE id = ? AND status = ?`,
		domain.PaymentRequestRejected, createdAt, requestID, domain.PaymentRequestPending); err != nil {
		return fmt.Errorf("reject payment request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("record rejection: %w", err)
	}
	return nil
}

func insertVote(tx *sql.Tx, requestID, userID string, decision domain.VoteDecision, createdAt string, now int64) error {
	var open int
	err := tx.QueryRow(`SELECT COUNT(1) FROM payment_requests WHERE id = ? AND status = ? AND expires_at > ?`,
		requestID, domain.PaymentRequestPending, now).Scan(&open)
	if err != nil {
		return fmt.Errorf("read payment request: %w", err)
	}
	if open == 0 {
		return storecontract.ErrConflict
	}
	result, err := tx.Exec(`INSERT OR IGNORE INTO payment_request_votes (request_id, user_id, decision, created_at) VALUES (?, ?, ?, ?)`,
		requestID, userID, decision, createdAt)
	if err != nil {
		return fmt.Errorf("insert vote: %w", err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("insert vote: %w", err)
	}
	if inserted == 0 {
		return storecontract.ErrDuplicate
	}
	return nil
}

// CancelPaymentRequest returns false when the request is no longer open.
func (s *PaymentRequestRepository) CancelPaymentRequest(requestID, decidedAt string, now int64) (bool, error) {
	return transition(s.db, "cancel payment request",
		`UPDATE payment_requests SET status = ?, decided_at = ? WHERE id = ? AND status = ? AND expires_at > ?`,
		domain.PaymentRequestCancelled, decidedAt, requestID, domain.PaymentRequestPending, now)
}

func (s *PaymentRequestRepository) AssignTransaction(requestID, transactionID string) error {
	return exec(s.db, "assign transaction",
		`UPDATE payment_requests SET transaction_id = ? WHERE id = ? AND status = ?`,
		transactionID, requestID, domain.PaymentRequestExecuting)
}

// SettleByTransaction returns the settled request id, or "" when no executing request owns the transaction.
func (s *PaymentRequestRepository) SettleByTransaction(transactionID string, status domain.PaymentRequestStatus, reason string) (string, error) {
	var requestID string
	err := s.db.QueryRow(`UPDATE payment_requests SET status = ?, reason = ? WHERE transaction_id = ? AND status = ? RETURNING id`,
		status, reason, transactionID, domain.PaymentRequestExecuting).Scan(&requestID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("settle payment request: %w", err)
	}
	return requestID, nil
}

func (s *PaymentRequestRepository) FailPaymentRequest(requestID, reason string) error {
	return exec(s.db, "fail payment request",
		`UPDATE payment_requests SET status = ?, reason = ? WHERE id = ? AND status = ?`,
		domain.PaymentRequestFailed, reason, requestID, domain.PaymentRequestExecuting)
}

// ExecutingPaymentRequests lists requests claimed for execution that never reached executed or failed.
func (s *PaymentRequestRepository) ExecutingPaymentRequests() ([]domain.PaymentRequest, error) {
	return s.paymentRequests(`WHERE payment_requests.status = ?`, domain.PaymentRequestExecuting)
}
