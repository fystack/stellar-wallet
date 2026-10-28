package service

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/mpc"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
)

type approvalFixture struct {
	*workspaceFixture
	team      domain.Workspace
	payments  *TransactionService
	approvals *PaymentRequestService
	chain     *paymentChainStub
	signer    *paymentSignerStub
}

// newApprovalFixture: owner and teammate (operator) can send; stranger joins as a read-only member.
func newApprovalFixture(t *testing.T, policy *ApprovalPolicyInput) *approvalFixture {
	t.Helper()
	f := newWorkspaceFixture(t)
	team := f.teamWorkspace(t)
	for _, user := range []string{"teammate", "stranger"} {
		if _, err := f.service.Accept(user, f.sessions[user], f.invite(t, team.ID, user+"@example.com")); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.service.ChangeRole("owner", team.ID, "teammate", domain.RoleOperator); err != nil {
		t.Fatal(err)
	}
	if err := f.store.CreateWallet(domain.Wallet{ID: "team-wallet", UserID: "owner", WorkspaceID: team.ID, Name: "Treasury", Chain: "stellar", Symbol: "XLM", Pubkey: "public-key", Status: domain.WalletReady}); err != nil {
		t.Fatal(err)
	}
	if policy != nil {
		if _, err := f.service.SetApprovalPolicy("owner", team.ID, *policy); err != nil {
			t.Fatal(err)
		}
	}
	stub := &paymentChainStub{resolved: chain.Resolved{Address: "resolved-address"}}
	signer := &paymentSignerStub{}
	hub := realtime.NewHub()
	payments := &TransactionService{wallets: f.store, transactions: sqlite.NewTransactionRepository(f.db), chain: stub, mpc: signer, cluster: paymentClusterStub(3), hub: hub}
	approvals := NewPaymentRequestService(f.store, f.workspaces, sqlite.NewPaymentRequestRepository(f.db), payments, hub, f.mailer, "http://localhost:5173/")
	payments.SetApprovals(approvals)
	f.mailer.sent = nil
	return &approvalFixture{workspaceFixture: f, team: team, payments: payments, approvals: approvals, chain: stub, signer: signer}
}

var oneApprovalAbove10 = &ApprovalPolicyInput{ThresholdXLM: "10", RequiredApprovals: 1, BelowThreshold: domain.BelowThresholdSend}

func (f *approvalFixture) pay(t *testing.T, userID string, input PaymentInput) PaymentResult {
	t.Helper()
	input.WalletID, input.To = "team-wallet", "destination"
	result, err := f.payments.Create(userID, input)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *approvalFixture) held(t *testing.T, userID, amount string) domain.PaymentRequest {
	t.Helper()
	result := f.pay(t, userID, PaymentInput{Amount: amount})
	if result.Request == nil {
		t.Fatalf("payment of %s was not held: %#v", amount, result.Transaction)
	}
	return *result.Request
}

func TestApprovalPolicyRoutesPayments(t *testing.T) {
	cases := []struct {
		name   string
		policy *ApprovalPolicyInput
		input  PaymentInput
		held   bool
	}{
		{"no policy sends immediately", nil, PaymentInput{Amount: "1000"}, false},
		{"at threshold sends", oneApprovalAbove10, PaymentInput{Amount: "10"}, false},
		{"one stroop above threshold is held", oneApprovalAbove10, PaymentInput{Amount: "10.0000001"}, true},
		{"below threshold held when policy says so", &ApprovalPolicyInput{ThresholdXLM: "10", RequiredApprovals: 1, BelowThreshold: domain.BelowThresholdApprove}, PaymentInput{Amount: "1"}, true},
		{"explicit XLM code is native", oneApprovalAbove10, PaymentInput{Amount: "1", AssetCode: "XLM"}, false},
		{"non-XLM asset is always held", oneApprovalAbove10, PaymentInput{Amount: "0.01", AssetCode: "USDC", AssetIssuer: "GISSUER"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, tc.policy)
			result := f.pay(t, "teammate", tc.input)
			if tc.held {
				if result.Request == nil || result.Transaction != nil || f.signer.calls != 0 {
					t.Fatalf("result = %#v, sign calls = %d", result, f.signer.calls)
				}
				if result.Request.Status != domain.PaymentRequestPending || result.Request.InitiatorEmail != "teammate@example.com" || result.Request.WalletName != "Treasury" {
					t.Fatalf("request = %#v", result.Request)
				}
				return
			}
			if result.Transaction == nil || result.Request != nil || f.signer.calls != 1 {
				t.Fatalf("result = %#v, sign calls = %d", result, f.signer.calls)
			}
		})
	}
}

func TestHeldPaymentEmailsOtherSendersOnly(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	request := f.held(t, "teammate", "20")
	f.approvals.notifications.Wait()
	if len(f.mailer.sent) != 1 || !strings.HasPrefix(f.mailer.sent[0], "owner@example.com\n") || !strings.Contains(f.mailer.sent[0], "/#/approvals/"+request.ID) {
		t.Fatalf("emails = %#v", f.mailer.sent)
	}
}

func TestApprovalExecutesHeldPayment(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	events := f.payments.hub.Subscribe("stranger")
	defer f.payments.hub.Unsubscribe("stranger", events)
	request := f.held(t, "teammate", "20")

	approved, err := f.approvals.Approve("owner", request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.PaymentRequestExecuting || approved.TransactionID == "" || approved.TransactionStatus != domain.TransactionSigning || f.signer.calls != 1 {
		t.Fatalf("approved = %#v, sign calls = %d", approved, f.signer.calls)
	}
	if len(approved.Votes) != 1 || approved.Votes[0].Email != "owner@example.com" || approved.Votes[0].Decision != domain.VoteApprove {
		t.Fatalf("votes = %#v", approved.Votes)
	}
	transaction, ok := f.payments.transactions.TransactionByID(approved.TransactionID)
	if !ok || transaction.UserID != "teammate" || transaction.Amount != "20" || f.chain.builtAmount != "20" {
		t.Fatalf("transaction = %#v", transaction)
	}
	if len(events) == 0 {
		t.Fatal("read-only member got no payment request events")
	}
	if _, err := f.approvals.Approve("owner", request.ID); err == nil {
		t.Fatal("second approval accepted")
	} else {
		assertCode(t, err, "payment_request_closed")
	}
	if f.signer.calls != 1 {
		t.Fatalf("sign calls = %d", f.signer.calls)
	}
}

func TestPaymentWaitsForEveryRequiredApproval(t *testing.T) {
	f := newApprovalFixture(t, nil)
	if err := f.service.ChangeRole("owner", f.team.ID, "stranger", domain.RoleOperator); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.SetApprovalPolicy("owner", f.team.ID, ApprovalPolicyInput{ThresholdXLM: "10", RequiredApprovals: 2, BelowThreshold: domain.BelowThresholdSend}); err != nil {
		t.Fatal(err)
	}
	request := f.held(t, "teammate", "20")
	first, err := f.approvals.Approve("owner", request.ID)
	if err != nil || first.Status != domain.PaymentRequestPending || f.signer.calls != 0 {
		t.Fatalf("after first approval = %#v, %v, sign calls = %d", first, err, f.signer.calls)
	}
	if _, err := f.approvals.Approve("owner", request.ID); err == nil {
		t.Fatal("duplicate vote accepted")
	} else {
		assertCode(t, err, "approval_already_voted")
	}
	second, err := f.approvals.Approve("stranger", request.ID)
	if err != nil || second.Status != domain.PaymentRequestExecuting || f.signer.calls != 1 || len(second.Votes) != 2 {
		t.Fatalf("after second approval = %#v, %v, sign calls = %d", second, err, f.signer.calls)
	}
}

func TestConcurrentFinalApprovalsSendOnce(t *testing.T) {
	f := newApprovalFixture(t, nil)
	if err := f.service.ChangeRole("owner", f.team.ID, "stranger", domain.RoleOperator); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.SetApprovalPolicy("owner", f.team.ID, *oneApprovalAbove10); err != nil {
		t.Fatal(err)
	}
	request := f.held(t, "teammate", "20")
	var wait sync.WaitGroup
	for _, approver := range []string{"owner", "stranger"} {
		wait.Add(1)
		go func() {
			defer wait.Done()
			f.approvals.Approve(approver, request.ID)
		}()
	}
	wait.Wait()
	stored, err := f.approvals.Get("owner", request.ID)
	if err != nil || stored.Status != domain.PaymentRequestExecuting || f.signer.calls != 1 {
		t.Fatalf("request = %#v, %v, sign calls = %d", stored, err, f.signer.calls)
	}
}

func TestApprovalRejections(t *testing.T) {
	cases := []struct {
		name   string
		userID string
		setup  func(t *testing.T, f *approvalFixture, request domain.PaymentRequest)
		code   string
	}{
		{"initiator cannot approve", "teammate", nil, "approval_self_not_allowed"},
		{"read-only member cannot approve", "stranger", nil, "approval_role_required"},
		{"outsider sees nothing", "outsider", nil, "payment_request_not_found"},
		{"rejected request", "owner", func(t *testing.T, f *approvalFixture, request domain.PaymentRequest) {
			if err := f.service.ChangeRole("owner", f.team.ID, "stranger", domain.RoleOperator); err != nil {
				t.Fatal(err)
			}
			if _, err := f.approvals.Reject("stranger", request.ID); err != nil {
				t.Fatal(err)
			}
		}, "payment_request_closed"},
		{"expired request", "owner", func(t *testing.T, f *approvalFixture, request domain.PaymentRequest) {
			if _, err := f.db.Exec(`UPDATE payment_requests SET expires_at = 1 WHERE id = ?`, request.ID); err != nil {
				t.Fatal(err)
			}
		}, "payment_request_closed"},
		{"MPC below quorum at final approval", "owner", func(_ *testing.T, f *approvalFixture, _ domain.PaymentRequest) {
			f.payments.cluster = paymentClusterStub(1)
		}, "mpc_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, oneApprovalAbove10)
			request := f.held(t, "teammate", "20")
			if tc.setup != nil {
				tc.setup(t, f, request)
			}
			_, err := f.approvals.Approve(tc.userID, request.ID)
			assertCode(t, err, tc.code)
			if f.signer.calls != 0 {
				t.Fatal("rejected approval dispatched signing")
			}
			if stored, _ := f.approvals.requests.PaymentRequestByID(request.ID); stored.VotedBy(tc.userID) {
				t.Fatal("rejected approval recorded a vote")
			}
		})
	}
}

func TestApprovalPolicyKeepsEnoughApprovers(t *testing.T) {
	cases := []struct {
		name string
		run  func(f *approvalFixture) error
		code string
	}{
		{"demote the only other sender", func(f *approvalFixture) error {
			return f.service.ChangeRole("owner", f.team.ID, "teammate", domain.RoleMember)
		}, "approval_policy_conflict"},
		{"remove the only other sender", func(f *approvalFixture) error {
			return f.service.RemoveMember("owner", f.team.ID, "teammate")
		}, "approval_policy_conflict"},
		{"remove a read-only member", func(f *approvalFixture) error {
			return f.service.RemoveMember("owner", f.team.ID, "stranger")
		}, ""},
		{"operator becomes owner", func(f *approvalFixture) error {
			return f.service.ChangeRole("owner", f.team.ID, "teammate", domain.RoleOwner)
		}, ""},
		{"demote after the policy is turned off", func(f *approvalFixture) error {
			if err := f.service.DisableApprovalPolicy("owner", f.team.ID); err != nil {
				return err
			}
			return f.service.ChangeRole("owner", f.team.ID, "teammate", domain.RoleMember)
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, oneApprovalAbove10)
			err := tc.run(f)
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			assertCode(t, err, tc.code)
			if workspace, err := f.workspaces.MemberWorkspace(f.team.ID, "teammate"); err != nil || workspace.Role != domain.RoleOperator {
				t.Fatalf("blocked change applied: %#v, %v", workspace, err)
			}
		})
	}
}

func TestHeldPaymentNeedsEnoughApprovers(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	if _, err := f.db.Exec(`UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?`, domain.RoleMember, f.team.ID, "teammate"); err != nil {
		t.Fatal(err)
	}
	_, err := f.payments.Create("owner", PaymentInput{WalletID: "team-wallet", To: "destination", Amount: "20"})
	assertCode(t, err, "approval_quorum_unavailable")
	if f.signer.calls != 0 || len(f.mailer.sent) != 0 {
		t.Fatal("unsatisfiable payment was dispatched or announced")
	}
}

func TestRejectAndCancel(t *testing.T) {
	cases := []struct {
		name   string
		run    func(f *approvalFixture, requestID string) (domain.PaymentRequest, error)
		status domain.PaymentRequestStatus
		code   string
	}{
		{"owner rejects", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			return f.approvals.Reject("owner", id)
		}, domain.PaymentRequestRejected, ""},
		{"initiator cannot reject", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			return f.approvals.Reject("teammate", id)
		}, "", "approval_self_not_allowed"},
		{"initiator cancels", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			return f.approvals.Cancel("teammate", id)
		}, domain.PaymentRequestCancelled, ""},
		{"owner cancels", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			return f.approvals.Cancel("owner", id)
		}, domain.PaymentRequestCancelled, ""},
		{"member cannot cancel", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			return f.approvals.Cancel("stranger", id)
		}, "", "payment_request_cancel_forbidden"},
		{"cancel twice", func(f *approvalFixture, id string) (domain.PaymentRequest, error) {
			if _, err := f.approvals.Cancel("teammate", id); err != nil {
				return domain.PaymentRequest{}, err
			}
			return f.approvals.Cancel("teammate", id)
		}, "", "payment_request_closed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, oneApprovalAbove10)
			request := f.held(t, "teammate", "20")
			result, err := tc.run(f, request.ID)
			if tc.code != "" {
				assertCode(t, err, tc.code)
			} else if err != nil || result.Status != tc.status || result.DecidedAt == "" {
				t.Fatalf("result = %#v, %v", result, err)
			}
			if f.signer.calls != 0 {
				t.Fatal("closed request dispatched signing")
			}
		})
	}
}

func TestFailedExecutionKeepsReason(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	request := f.held(t, "teammate", "20")
	f.chain.buildErr = errors.New("source account not found")
	approved, err := f.approvals.Approve("owner", request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.PaymentRequestFailed || !strings.HasPrefix(approved.Reason, "Unable to prepare payment") || f.signer.calls != 0 {
		t.Fatalf("approved = %#v", approved)
	}
}

func TestListPaymentRequests(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	open := f.held(t, "teammate", "20")
	closed := f.held(t, "teammate", "30")
	if _, err := f.approvals.Cancel("teammate", closed.ID); err != nil {
		t.Fatal(err)
	}
	pending, err := f.approvals.List("stranger", f.team.ID, PaymentRequestsOpen, PageInput{})
	if err != nil || len(pending.Items) != 1 || pending.Items[0].ID != open.ID || pending.Next != "" {
		t.Fatalf("pending = %#v, %v", pending, err)
	}
	history, err := f.approvals.List("stranger", f.team.ID, PaymentRequestsClosed, PageInput{})
	if err != nil || len(history.Items) != 1 || history.Items[0].ID != closed.ID || history.Next != "" {
		t.Fatalf("history = %#v, %v", history, err)
	}
	rejections := []struct {
		name   string
		userID string
		filter string
		page   PageInput
		code   string
	}{
		{"outsider", "outsider", PaymentRequestsOpen, PageInput{}, "workspace_not_found"},
		{"unknown filter", "owner", "everything", PageInput{}, "invalid_request"},
		{"page too large", "owner", PaymentRequestsClosed, PageInput{Limit: maxPageSize + 1}, "invalid_request"},
		{"negative page size", "owner", PaymentRequestsClosed, PageInput{Limit: -1}, "invalid_request"},
		{"malformed cursor", "owner", PaymentRequestsClosed, PageInput{Cursor: "not a cursor"}, "invalid_request"},
	}
	for _, tc := range rejections {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.approvals.List(tc.userID, f.team.ID, tc.filter, tc.page)
			assertCode(t, err, tc.code)
		})
	}
}

func TestPaymentRequestHistoryPages(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	var cancelled []string
	for range 5 {
		request := f.held(t, "teammate", "20")
		if _, err := f.approvals.Cancel("teammate", request.ID); err != nil {
			t.Fatal(err)
		}
		cancelled = append(cancelled, request.ID)
	}
	seen := map[string]bool{}
	page := PageInput{Limit: 2}
	for pages := 1; ; pages++ {
		result, err := f.approvals.List("owner", f.team.ID, PaymentRequestsClosed, page)
		if err != nil || len(result.Items) == 0 || len(result.Items) > 2 {
			t.Fatalf("page %d = %#v, %v", pages, result, err)
		}
		for _, request := range result.Items {
			if seen[request.ID] {
				t.Fatalf("request %s returned twice", request.ID)
			}
			seen[request.ID] = true
		}
		if result.Next == "" {
			if pages != 3 {
				t.Fatalf("pages = %d, want 3", pages)
			}
			break
		}
		page.Cursor = result.Next
	}
	for _, id := range cancelled {
		if !seen[id] {
			t.Fatalf("request %s never returned", id)
		}
	}
}

func TestReconcileSettlesClaimedRequests(t *testing.T) {
	cases := []struct {
		name              string
		transactionStatus domain.TransactionStatus
		want              domain.PaymentRequestStatus
	}{
		{"transaction never saved", "", domain.PaymentRequestFailed},
		{"transaction still signing", domain.TransactionSigning, domain.PaymentRequestExecuting},
		{"transaction confirmed", domain.TransactionConfirmed, domain.PaymentRequestExecuted},
		{"transaction failed", domain.TransactionFailed, domain.PaymentRequestFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, oneApprovalAbove10)
			request := f.held(t, "teammate", "20")
			if _, err := f.db.Exec(`UPDATE payment_requests SET status = ?, transaction_id = ? WHERE id = ?`, domain.PaymentRequestExecuting, "tx-1", request.ID); err != nil {
				t.Fatal(err)
			}
			if tc.transactionStatus != "" {
				if err := f.payments.transactions.CreateTransaction(domain.Transaction{ID: "tx-1", WalletID: "team-wallet", UserID: "teammate", Type: domain.TransactionOut, Status: tc.transactionStatus}); err != nil {
					t.Fatal(err)
				}
			}
			f.approvals.ReconcileExecuting()
			stored, err := f.approvals.Get("owner", request.ID)
			if err != nil || stored.Status != tc.want {
				t.Fatalf("stored = %#v, %v", stored, err)
			}
		})
	}
}

func TestApprovedPaymentSettlesWithItsTransaction(t *testing.T) {
	cases := []struct {
		name       string
		result     func(transactionID string) mpc.SignResult
		submitErr  error
		want       domain.PaymentRequestStatus
		wantReason string
	}{
		{"broadcast confirms", func(id string) mpc.SignResult {
			return mpc.SignResult{TransactionID: id, Successful: true, Signature: []byte("signature")}
		}, nil, domain.PaymentRequestExecuted, ""},
		{"signing fails", func(id string) mpc.SignResult {
			return mpc.SignResult{TransactionID: id, Error: "node rejected"}
		}, nil, domain.PaymentRequestFailed, "node rejected"},
		{"broadcast fails", func(id string) mpc.SignResult {
			return mpc.SignResult{TransactionID: id, Successful: true, Signature: []byte("signature")}
		}, errors.New("tx_bad_seq"), domain.PaymentRequestFailed, "broadcast: tx_bad_seq"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newApprovalFixture(t, oneApprovalAbove10)
			request := f.held(t, "teammate", "20")
			approved, err := f.approvals.Approve("owner", request.ID)
			if err != nil {
				t.Fatal(err)
			}
			f.chain.submitErr = tc.submitErr
			f.payments.OnSign(tc.result(approved.TransactionID))
			stored, err := f.approvals.Get("owner", request.ID)
			if err != nil || stored.Status != tc.want || stored.Reason != tc.wantReason {
				t.Fatalf("stored = %#v, %v", stored, err)
			}
		})
	}
}

func TestApprovalsCountOnlyCurrentSenders(t *testing.T) {
	f := newApprovalFixture(t, nil)
	if err := f.service.ChangeRole("owner", f.team.ID, "stranger", domain.RoleOperator); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.SetApprovalPolicy("owner", f.team.ID, ApprovalPolicyInput{ThresholdXLM: "10", RequiredApprovals: 2, BelowThreshold: domain.BelowThresholdSend}); err != nil {
		t.Fatal(err)
	}
	request := f.held(t, "teammate", "20")
	if _, err := f.approvals.Approve("stranger", request.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?`, domain.RoleMember, f.team.ID, "stranger"); err != nil {
		t.Fatal(err)
	}
	approved, err := f.approvals.Approve("owner", request.ID)
	if err != nil || approved.Status != domain.PaymentRequestPending || approved.Approvals() != 1 || f.signer.calls != 0 {
		t.Fatalf("approved = %#v, %v, sign calls = %d", approved, err, f.signer.calls)
	}
}

func TestPaymentFailsWhenInitiatorCanNoLongerSend(t *testing.T) {
	f := newApprovalFixture(t, oneApprovalAbove10)
	request := f.held(t, "teammate", "20")
	if _, err := f.db.Exec(`UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?`, domain.RoleMember, f.team.ID, "teammate"); err != nil {
		t.Fatal(err)
	}
	approved, err := f.approvals.Approve("owner", request.ID)
	if err != nil || approved.Status != domain.PaymentRequestFailed || f.signer.calls != 0 {
		t.Fatalf("approved = %#v, %v, sign calls = %d", approved, err, f.signer.calls)
	}
}
