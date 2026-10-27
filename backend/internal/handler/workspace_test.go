package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository/sqlite"
	"stellar-wallet-backend/internal/service"
)

type walletProvisionerStub struct{}

func (walletProvisionerStub) Provision(userID, workspaceID, name, _ string) (domain.Wallet, error) {
	return domain.Wallet{ID: "wallet-" + userID, UserID: userID, WorkspaceID: workspaceID, Name: name}, nil
}

func TestWorkspaceInviteHTTPContract(t *testing.T) {
	manager, accounts, repo, db := newAuthFixture(t)
	workspaces := service.NewWorkspaceService(sqlite.NewWorkspaceRepository(db), repo, walletProvisionerStub{}, nil, "http://localhost")
	router := NewRouter(Dependencies{AuthRateLimit: 30, Auth: manager, AuthService: accounts, WorkspaceService: workspaces}, "http://localhost")
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
		req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"email":"`+email+`","password":"password123"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Protection", "1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("register %s = %d %s", email, w.Code, w.Body)
		}
		return w.Result().Cookies()[0]
	}
	data := func(body map[string]any) map[string]any { return body["data"].(map[string]any) }
	errorCode := func(body map[string]any) any { return body["error"].(map[string]any)["code"] }

	owner, teammate := register("owner@example.com"), register("teammate@example.com")

	status, body := request("POST", "/workspaces", `{"name":"Treasury"}`, owner)
	if status != http.StatusCreated {
		t.Fatalf("create workspace = %d %#v", status, body)
	}
	workspaceID := data(body)["workspace"].(map[string]any)["id"].(string)

	status, body = request("POST", "/workspaces/"+workspaceID+"/invites", `{"email":"teammate@example.com"}`, owner)
	if status != http.StatusCreated || data(body)["emailSent"] != false {
		t.Fatalf("invite = %d %#v", status, body)
	}
	_, token, _ := strings.Cut(data(body)["link"].(string), "/#/invite/")

	status, body = request("GET", "/invites/"+token, "", nil)
	if status != http.StatusOK || data(body)["status"] != "pending" || data(body)["workspaceName"] != "Treasury" || data(body)["inviteId"] == "" {
		t.Fatalf("public preview = %d %#v", status, body)
	}
	if status, body = request("POST", "/invites/"+token+"/accept", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous accept = %d %#v", status, body)
	}
	if status, body = request("POST", "/invites/"+token+"/accept", "", owner); status != http.StatusForbidden || errorCode(body) != "invite_email_mismatch" {
		t.Fatalf("wrong account accept = %d %#v", status, body)
	}
	if status, body = request("POST", "/invites/"+token+"/accept", "", teammate); status != http.StatusOK || data(body)["wallet"] != nil {
		t.Fatalf("accept = %d %#v", status, body)
	}

	if status, body = request("GET", "/workspaces", "", teammate); status != http.StatusOK || data(body)["activeId"] != workspaceID {
		t.Fatalf("teammate workspaces = %d %#v", status, body)
	}
	status, body = request("GET", "/workspaces/"+workspaceID+"/members", "", teammate)
	if status != http.StatusOK || len(data(body)["members"].([]any)) != 2 {
		t.Fatalf("members = %d %#v", status, body)
	}
	if status, body = request("POST", "/workspaces/"+workspaceID+"/invites", `{"email":"new@example.com"}`, teammate); status != http.StatusForbidden || errorCode(body) != "workspace_owner_required" {
		t.Fatalf("member invite = %d %#v", status, body)
	}

	policyPath := "/workspaces/" + workspaceID + "/approval-policy"
	status, body = request("GET", policyPath, "", teammate)
	if status != http.StatusOK || data(body)["policy"] != nil || data(body)["maxApprovals"] != float64(0) {
		t.Fatalf("empty policy = %d %#v", status, body)
	}
	policy := `{"thresholdXlm":"10","requiredApprovals":1,"belowThreshold":"send"}`
	if status, body = request("PUT", policyPath, policy, owner); status != http.StatusBadRequest || errorCode(body) != "approval_policy_invalid" {
		t.Fatalf("policy without approvers = %d %#v", status, body)
	}
	status, body = request("GET", "/workspaces/"+workspaceID+"/members", "", owner)
	memberIDs := map[string]string{}
	for _, member := range data(body)["members"].([]any) {
		member := member.(map[string]any)
		memberIDs[member["email"].(string)] = member["userId"].(string)
	}
	teammatePath := "/workspaces/" + workspaceID + "/members/" + memberIDs["teammate@example.com"]
	if status, body = request("PATCH", teammatePath, `{"role":"operator"}`, teammate); status != http.StatusForbidden || errorCode(body) != "workspace_owner_required" {
		t.Fatalf("member changes role = %d %#v", status, body)
	}
	if status, body = request("PATCH", teammatePath, `{"role":"admin"}`, owner); status != http.StatusBadRequest || errorCode(body) != "invalid_role" {
		t.Fatalf("invalid role = %d %#v", status, body)
	}
	if status, body = request("PATCH", teammatePath, `{"role":"operator"}`, owner); status != http.StatusNoContent {
		t.Fatalf("promote operator = %d %#v", status, body)
	}
	ownerPath := "/workspaces/" + workspaceID + "/members/" + memberIDs["owner@example.com"]
	if status, body = request("PATCH", ownerPath, `{"role":"member"}`, owner); status != http.StatusConflict || errorCode(body) != "last_owner" {
		t.Fatalf("demote last owner = %d %#v", status, body)
	}
	status, body = request("PUT", policyPath, policy, owner)
	if status != http.StatusOK || data(body)["policy"].(map[string]any)["thresholdXlm"] != "10" || data(body)["maxApprovals"] != float64(1) {
		t.Fatalf("set policy = %d %#v", status, body)
	}
	if status, body = request("PUT", policyPath, policy, teammate); status != http.StatusForbidden || errorCode(body) != "workspace_owner_required" {
		t.Fatalf("operator sets policy = %d %#v", status, body)
	}
	if status, body = request("DELETE", policyPath, "", owner); status != http.StatusNoContent {
		t.Fatalf("disable policy = %d %#v", status, body)
	}
	if status, body = request("GET", policyPath, "", owner); status != http.StatusOK || data(body)["policy"] != nil {
		t.Fatalf("disabled policy = %d %#v", status, body)
	}

	request("POST", "/workspaces/"+workspaceID+"/invites", `{"email":"unverified@example.com"}`, owner)
	unverified := register("unverified@example.com")
	status, body = request("GET", "/invitations", "", unverified)
	if status != http.StatusOK || data(body)["verified"] != false || len(data(body)["invites"].([]any)) != 0 {
		t.Fatalf("unverified invitations = %d %#v", status, body)
	}
	if status, body = request("POST", "/invitations/missing/accept", "", unverified); status != http.StatusNotFound || errorCode(body) != "invite_not_found" {
		t.Fatalf("accept unknown invitation = %d %#v", status, body)
	}
	if status, body = request("POST", "/invitations/missing/decline", "", unverified); status != http.StatusNotFound || errorCode(body) != "invite_not_found" {
		t.Fatalf("decline unknown invitation = %d %#v", status, body)
	}
}
