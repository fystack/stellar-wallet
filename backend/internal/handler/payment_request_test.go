package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/realtime"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"
)

func TestPaymentApprovalHTTPContract(t *testing.T) {
	manager, accounts, repo, db := newAuthFixture(t)
	store := sqlite.NewStore(db)
	workspaceRepo := sqlite.NewWorkspaceRepository(db)
	hub := realtime.NewHub()
	workspaces := service.NewWorkspaceService(workspaceRepo, repo, nil, "http://localhost")
	transactions := service.NewTransactionService(store, sqlite.NewTransactionRepository(db), &paymentChainStub{resolved: chain.Resolved{Address: "resolved-address"}}, &paymentClusterStub{online: 3}, hub)
	signer := &paymentSignerStub{}
	transactions.SetMPC(signer)
	approvals := service.NewPaymentRequestService(store, workspaceRepo, sqlite.NewPaymentRequestRepository(db), transactions, hub, nil, "http://localhost")
	transactions.SetApprovals(approvals)
	router := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: accounts, WorkspaceService: workspaces, TransactionService: transactions, PaymentRequestService: approvals, Hub: hub}, "http://localhost")

	request := func(method, path, body string, cookie *http.Cookie) (int, map[string]any) {
		req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Protection", "1")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		var decoded map[string]any
		json.Unmarshal(w.Body.Bytes(), &decoded)
		return w.Code, decoded
	}
	register := func(email string) *http.Cookie {
		status, body := request("POST", "/auth/register", `{"email":"`+email+`","password":"password123"}`, nil)
		if status != http.StatusCreated {
			t.Fatalf("register %s = %d %#v", email, status, body)
		}
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"`+email+`","password":"password123"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Protection", "1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Result().Cookies()[0]
	}
	data := func(body map[string]any) map[string]any { return body["data"].(map[string]any) }
	errorCode := func(body map[string]any) any { return body["error"].(map[string]any)["code"] }

	owner, operator := register("owner@example.com"), register("operator@example.com")
	_, body := request("POST", "/workspaces", `{"name":"Treasury"}`, owner)
	workspaceID := data(body)["workspace"].(map[string]any)["id"].(string)
	_, body = request("POST", "/workspaces/"+workspaceID+"/invites", `{"email":"operator@example.com","role":"operator"}`, owner)
	_, token, _ := strings.Cut(data(body)["link"].(string), "/#/invite/")
	if status, body := request("POST", "/invites/"+token+"/accept", "", operator); status != http.StatusOK {
		t.Fatalf("accept = %d %#v", status, body)
	}
	if status, body := request("PUT", "/workspaces/"+workspaceID+"/approval-policy", `{"thresholdXlm":"10","requiredApprovals":1,"belowThreshold":"send"}`, owner); status != http.StatusOK {
		t.Fatalf("set policy = %d %#v", status, body)
	}
	if err := store.CreateWallet(domain.Wallet{ID: "team-wallet", UserID: "owner", WorkspaceID: workspaceID, Name: "Treasury", Chain: "stellar", Symbol: "XLM", Status: domain.WalletReady}); err != nil {
		t.Fatal(err)
	}

	status, body := request("POST", "/transactions", `{"wallet_id":"team-wallet","to":"destination","amount":"20"}`, operator)
	if status != http.StatusAccepted || data(body)["transaction"] != nil {
		t.Fatalf("held payment = %d %#v", status, body)
	}
	held := data(body)["request"].(map[string]any)
	requestID := held["id"].(string)
	if held["status"] != "pending" || held["requiredApprovals"] != float64(1) || held["initiatorEmail"] != "operator@example.com" {
		t.Fatalf("held request = %#v", held)
	}

	status, body = request("GET", "/workspaces/"+workspaceID+"/payment-requests?status=pending", "", owner)
	if status != http.StatusOK || len(body["data"].([]any)) != 1 || body["meta"].(map[string]any)["has_more"] != false {
		t.Fatalf("pending list = %d %#v", status, body)
	}
	status, body = request("GET", "/workspaces/"+workspaceID+"/payment-requests?status=history&limit=1", "", owner)
	if meta := body["meta"].(map[string]any); status != http.StatusOK || len(body["data"].([]any)) != 0 || meta["next_cursor"] != nil || meta["has_more"] != false {
		t.Fatalf("history page = %d %#v", status, body)
	}
	if status, body = request("GET", "/workspaces/"+workspaceID+"/payment-requests?status=history&limit=many", "", owner); status != http.StatusBadRequest || errorCode(body) != "invalid_request" {
		t.Fatalf("bad page size = %d %#v", status, body)
	}
	if status, body = request("POST", "/payment-requests/"+requestID+"/approve", "", operator); status != http.StatusForbidden || errorCode(body) != "approval_self_not_allowed" {
		t.Fatalf("self approval = %d %#v", status, body)
	}
	status, body = request("POST", "/payment-requests/"+requestID+"/approve", "", owner)
	if status != http.StatusOK || data(body)["status"] != "executing" || data(body)["transactionId"] == nil || signer.transactionID == "" {
		t.Fatalf("approve = %d %#v", status, body)
	}
	if status, body = request("POST", "/payment-requests/"+requestID+"/cancel", "", operator); status != http.StatusConflict || errorCode(body) != "payment_request_closed" {
		t.Fatalf("cancel executing = %d %#v", status, body)
	}
	status, body = request("GET", "/payment-requests/"+requestID, "", operator)
	if status != http.StatusOK || len(data(body)["votes"].([]any)) != 1 {
		t.Fatalf("get request = %d %#v", status, body)
	}
	if status, body = request("GET", "/payment-requests/missing", "", owner); status != http.StatusNotFound || errorCode(body) != "payment_request_not_found" {
		t.Fatalf("missing request = %d %#v", status, body)
	}

	status, body = request("POST", "/transactions", `{"wallet_id":"team-wallet","to":"destination","amount":"5"}`, operator)
	if status != http.StatusCreated || data(body)["transaction"] == nil {
		t.Fatalf("below threshold payment = %d %#v", status, body)
	}
}
