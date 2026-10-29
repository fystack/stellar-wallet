package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mail"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
	"github.com/stellar/go/amount"
)

const (
	paymentRequestTTL     = 7 * 24 * time.Hour
	PaymentRequestsOpen   = "pending"
	PaymentRequestsClosed = "history"
)

type PaymentRequestService struct {
	wallets       repository.Store
	workspaces    *sqlite.WorkspaceRepository
	requests      *sqlite.PaymentRequestRepository
	payments      *TransactionService
	hub           *realtime.Hub
	mailer        Mailer
	frontendURL   string
	notifications sync.WaitGroup
}

func NewPaymentRequestService(wallets repository.Store, workspaces *sqlite.WorkspaceRepository, requests *sqlite.PaymentRequestRepository, payments *TransactionService, hub *realtime.Hub, mailer Mailer, frontendURL string) *PaymentRequestService {
	return &PaymentRequestService{wallets: wallets, workspaces: workspaces, requests: requests, payments: payments, hub: hub, mailer: mailer, frontendURL: strings.TrimRight(frontendURL, "/")}
}

func needsApproval(policy domain.ApprovalPolicy, payment preparedPayment) (bool, error) {
	if !payment.native() {
		return true, nil
	}
	paymentStroops, err := amount.ParseInt64(payment.Amount)
	if err != nil {
		return false, fmt.Errorf("parse payment amount: %w", err)
	}
	thresholdStroops, err := amount.ParseInt64(policy.ThresholdXLM)
	if err != nil {
		return false, fmt.Errorf("parse approval threshold: %w", err)
	}
	if paymentStroops > thresholdStroops {
		return true, nil
	}
	return policy.BelowThreshold == domain.BelowThresholdApprove, nil
}

// hold returns nil when the payment can be sent right away.
func (s *PaymentRequestService) hold(wallet domain.Wallet, initiatorID string, payment preparedPayment) (*domain.PaymentRequest, error) {
	policy, err := s.workspaces.ApprovalPolicy(wallet.WorkspaceID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	required, err := needsApproval(policy, payment)
	if err != nil || !required {
		return nil, err
	}
	approvers, err := maxApprovals(s.workspaces, wallet.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if approvers < policy.RequiredApprovals {
		return nil, apperror.ApprovalQuorumUnavailable.New()
	}
	now := time.Now()
	request := domain.PaymentRequest{
		ID:                uuid.NewString(),
		WorkspaceID:       wallet.WorkspaceID,
		WalletID:          wallet.ID,
		InitiatorID:       initiatorID,
		Destination:       payment.Destination,
		Amount:            payment.Amount,
		AssetCode:         payment.AssetCode,
		AssetIssuer:       payment.AssetIssuer,
		Symbol:            payment.Symbol,
		Memo:              payment.Memo,
		MemoType:          payment.MemoType,
		RequiredApprovals: policy.RequiredApprovals,
		Status:            domain.PaymentRequestPending,
		CreatedAt:         now.UTC().Format(time.RFC3339),
		ExpiresAt:         now.Add(paymentRequestTTL).Unix(),
	}
	if err := s.requests.CreatePaymentRequest(request); err != nil {
		return nil, err
	}
	stored, err := s.requests.PaymentRequestByID(request.ID)
	if err != nil {
		return nil, err
	}
	s.publish(stored)
	s.notifications.Add(1)
	go func() {
		defer s.notifications.Done()
		s.notifyApprovers(stored)
	}()
	return &stored, nil
}

// List returns every open request; history comes a page at a time.
func (s *PaymentRequestService) List(userID, workspaceID, filter string, input PageInput) (Page[domain.PaymentRequest], error) {
	before, limit, err := input.resolve()
	if err != nil {
		return Page[domain.PaymentRequest]{}, err
	}
	if _, err := memberWorkspace(s.workspaces, workspaceID, userID); err != nil {
		return Page[domain.PaymentRequest]{}, err
	}
	now := time.Now().Unix()
	var page Page[domain.PaymentRequest]
	switch filter {
	case PaymentRequestsOpen, "":
		page.Items, err = s.requests.OpenPaymentRequests(workspaceID, now)
	case PaymentRequestsClosed:
		var requests []domain.PaymentRequest
		requests, err = s.requests.ClosedPaymentRequests(workspaceID, now, before, limit+1)
		page = pageOf(requests, limit, func(request domain.PaymentRequest) repository.Cursor {
			return repository.Cursor{CreatedAt: request.CreatedAt, ID: request.ID}
		})
	default:
		return page, apperror.InvalidPaymentRequestFilter.New()
	}
	if err != nil {
		return Page[domain.PaymentRequest]{}, err
	}
	for index := range page.Items {
		page.Items[index] = withEffectiveStatus(page.Items[index], now)
	}
	return page, nil
}

func (s *PaymentRequestService) Get(userID, requestID string) (domain.PaymentRequest, error) {
	request, _, err := s.visibleRequest(userID, requestID)
	return request, err
}

func (s *PaymentRequestService) Approve(userID, requestID string) (domain.PaymentRequest, error) {
	request, err := s.votableRequest(userID, requestID)
	if err != nil {
		return domain.PaymentRequest{}, err
	}
	if request.Approvals()+1 >= request.RequiredApprovals {
		if err := s.payments.requireSigning(); err != nil {
			return domain.PaymentRequest{}, err
		}
	}
	now := time.Now()
	claimed, err := s.requests.RecordApproval(request.ID, userID, now.UTC().Format(time.RFC3339), now.Unix())
	if err := voteError(err); err != nil {
		return domain.PaymentRequest{}, err
	}
	if claimed {
		s.execute(request)
	}
	return s.refreshed(request.ID)
}

func (s *PaymentRequestService) Reject(userID, requestID string) (domain.PaymentRequest, error) {
	request, err := s.votableRequest(userID, requestID)
	if err != nil {
		return domain.PaymentRequest{}, err
	}
	now := time.Now()
	if err := voteError(s.requests.RecordRejection(request.ID, userID, now.UTC().Format(time.RFC3339), now.Unix())); err != nil {
		return domain.PaymentRequest{}, err
	}
	return s.refreshed(request.ID)
}

func (s *PaymentRequestService) Cancel(userID, requestID string) (domain.PaymentRequest, error) {
	request, role, err := s.visibleRequest(userID, requestID)
	if err != nil {
		return domain.PaymentRequest{}, err
	}
	if request.InitiatorID != userID && role != domain.RoleOwner {
		return domain.PaymentRequest{}, apperror.PaymentRequestCancelForbidden.New()
	}
	now := time.Now()
	cancelled, err := s.requests.CancelPaymentRequest(request.ID, now.UTC().Format(time.RFC3339), now.Unix())
	if err != nil {
		return domain.PaymentRequest{}, err
	}
	if !cancelled {
		return domain.PaymentRequest{}, apperror.PaymentRequestClosed.New()
	}
	return s.refreshed(request.ID)
}

// ReconcileExecuting settles requests a previous run claimed from the state of their transaction.
func (s *PaymentRequestService) ReconcileExecuting() {
	requests, err := s.requests.ExecutingPaymentRequests()
	if err != nil {
		logError("reconcile executing payment requests", err)
		return
	}
	for _, request := range requests {
		transaction, saved := s.payments.transactions.TransactionByID(request.TransactionID)
		if !saved {
			s.fail(request.ID, "The server restarted before this payment was sent. Start the payment again.")
			continue
		}
		s.settle(transaction)
	}
}

// settle closes the executing request that owns a transaction once that transaction confirms or fails.
func (s *PaymentRequestService) settle(transaction domain.Transaction) {
	var status domain.PaymentRequestStatus
	switch transaction.Status {
	case domain.TransactionConfirmed:
		status = domain.PaymentRequestExecuted
	case domain.TransactionFailed:
		status = domain.PaymentRequestFailed
	default:
		return
	}
	requestID, err := s.requests.SettleByTransaction(transaction.ID, status, transaction.Error)
	if err != nil {
		logError("settle payment request tx="+transaction.ID, err)
		return
	}
	if requestID != "" {
		s.publishByID(requestID)
	}
}

// execute runs only for the caller that claimed the request, so a payment is sent at most once.
func (s *PaymentRequestService) execute(request domain.PaymentRequest) {
	wallet, ok := s.wallets.WalletByID(request.WalletID)
	if !ok || wallet.Status != domain.WalletReady {
		s.fail(request.ID, "The wallet is no longer available.")
		return
	}
	role, err := s.wallets.WorkspaceRole(request.WorkspaceID, request.InitiatorID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		logError("read initiator role payment request="+request.ID, err)
	}
	if !role.CanSend() {
		s.fail(request.ID, "The person who started this payment can no longer send from this workspace.")
		return
	}
	// The transaction id is recorded before signing starts, so a restart can tell a sent payment from an unsent one.
	transactionID := uuid.NewString()
	if err := s.requests.AssignTransaction(request.ID, transactionID); err != nil {
		logError("assign transaction payment request="+request.ID, err)
		s.fail(request.ID, "The payment could not be sent.")
		return
	}
	payment := preparedPayment{
		Destination: request.Destination,
		Amount:      request.Amount,
		AssetCode:   request.AssetCode,
		AssetIssuer: request.AssetIssuer,
		Symbol:      request.Symbol,
		Memo:        request.Memo,
		MemoType:    request.MemoType,
	}
	_, err = s.payments.sendPayment(wallet, request.InitiatorID, payment, transactionID)
	if err != nil {
		reason := "The payment could not be sent."
		var failure *apperror.Error
		if errors.As(err, &failure) {
			reason = failure.Message
		}
		logError("execute payment request="+request.ID, err)
		s.fail(request.ID, reason)
		return
	}
	s.publishByID(request.ID)
}

func (s *PaymentRequestService) fail(requestID, reason string) {
	logError("fail payment request="+requestID, s.requests.FailPaymentRequest(requestID, reason))
	s.publishByID(requestID)
}

// visibleRequest reports requests outside the user's workspaces as not found.
func (s *PaymentRequestService) visibleRequest(userID, requestID string) (domain.PaymentRequest, domain.WorkspaceRole, error) {
	request, err := s.requests.PaymentRequestByID(requestID)
	if errors.Is(err, repository.ErrNotFound) {
		return request, "", apperror.PaymentRequestNotFound.New()
	}
	if err != nil {
		return request, "", err
	}
	workspace, err := s.workspaces.MemberWorkspace(request.WorkspaceID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return domain.PaymentRequest{}, "", apperror.PaymentRequestNotFound.New()
	}
	if err != nil {
		return domain.PaymentRequest{}, "", err
	}
	return withEffectiveStatus(request, time.Now().Unix()), workspace.Role, nil
}

func (s *PaymentRequestService) votableRequest(userID, requestID string) (domain.PaymentRequest, error) {
	request, role, err := s.visibleRequest(userID, requestID)
	if err != nil {
		return request, err
	}
	switch {
	case !role.CanSend():
		return request, apperror.ApprovalRoleRequired.New()
	case request.InitiatorID == userID:
		return request, apperror.ApprovalSelfNotAllowed.New()
	case request.Status != domain.PaymentRequestPending:
		return request, apperror.PaymentRequestClosed.New()
	case request.VotedBy(userID):
		return request, apperror.ApprovalAlreadyVoted.New()
	}
	return request, nil
}

func voteError(err error) error {
	switch {
	case errors.Is(err, repository.ErrConflict):
		return apperror.PaymentRequestClosed.New()
	case errors.Is(err, repository.ErrDuplicate):
		return apperror.ApprovalAlreadyVoted.New()
	}
	return err
}

func (s *PaymentRequestService) refreshed(requestID string) (domain.PaymentRequest, error) {
	request, err := s.requests.PaymentRequestByID(requestID)
	if err != nil {
		return request, err
	}
	request = withEffectiveStatus(request, time.Now().Unix())
	s.publish(request)
	return request, nil
}

func withEffectiveStatus(request domain.PaymentRequest, now int64) domain.PaymentRequest {
	if request.Status == domain.PaymentRequestPending && request.ExpiresAt <= now {
		request.Status = domain.PaymentRequestExpired
	}
	return request
}

func (s *PaymentRequestService) publishByID(requestID string) {
	_, err := s.refreshed(requestID)
	logError("read payment request="+requestID, err)
}

func (s *PaymentRequestService) publish(request domain.PaymentRequest) {
	userIDs, err := s.wallets.WorkspaceMemberIDs(request.WorkspaceID)
	if err != nil {
		logError("list event audience payment request="+request.ID, err)
		return
	}
	for _, userID := range userIDs {
		s.hub.Publish(userID, "payment_request", request)
	}
}

// notifyApprovers emails every sender except the initiator; email failures never block the request.
func (s *PaymentRequestService) notifyApprovers(request domain.PaymentRequest) {
	if s.mailer == nil {
		return
	}
	members, err := s.workspaces.Members(request.WorkspaceID)
	if err != nil {
		logError("list approvers payment request="+request.ID, err)
		return
	}
	workspaceName, err := s.workspaces.WorkspaceName(request.WorkspaceID)
	if err != nil {
		logError("read workspace payment request="+request.ID, err)
		return
	}
	message := mail.Message{
		Subject: fmt.Sprintf("Approval needed: %s %s from %s", request.Amount, request.Symbol, request.WalletName),
		Heading: fmt.Sprintf("Approve %s %s?", request.Amount, request.Symbol),
		Intro:   fmt.Sprintf("%s wants to send a payment from the %s workspace. It needs %d approval(s) before it is signed.", request.InitiatorEmail, workspaceName, request.RequiredApprovals),
		Details: []mail.Detail{
			{Label: "Amount", Value: request.Amount + " " + request.Symbol},
			{Label: "From", Value: request.WalletName},
			{Label: "To", Value: request.Destination, Mono: true},
		},
		ActionLabel: "Review request",
		ActionURL:   s.frontendURL + "/#/approvals/" + request.ID,
		Note:        "The request expires in 7 days if it is not approved.",
	}
	for _, member := range members {
		if member.UserID == request.InitiatorID || !member.Role.CanSend() {
			continue
		}
		logError("send approval email payment request="+request.ID, s.mailer.Send(member.Email, message))
	}
}
