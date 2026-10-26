package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/repository/sqlite"
)

func TestAuthEnvelopesAndDatabaseFailures(t *testing.T) {
	db, err := sqlite.Open(t.TempDir() + "/auth.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(Dependencies{Store: sqlite.NewStore(db), Auth: auth.New([]byte("test-secret"), time.Hour)})
	r := s.Router("http://localhost")
	post := func(path, body string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/v1/auth/"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return w.Code, response
	}
	credentials := `{"email":"owner@example.com","password":"password123"}`
	status, body := post("register", credentials)
	data, ok := body["data"].(map[string]any)
	if status != 201 || !ok || data["access_token"] == "" || data["email"] != "owner@example.com" {
		t.Fatalf("register = %d %#v", status, body)
	}
	status, body = post("login", credentials)
	if status != 200 || body["data"] == nil {
		t.Fatalf("login = %d %#v", status, body)
	}
	for _, tt := range []struct {
		path, body string
		status     int
		code       string
	}{
		{"register", credentials, 409, "email_already_registered"},
		{"login", `{"email":"unknown@example.com","password":"password123"}`, 401, "invalid_credentials"},
		{"login", `{"email":"owner@example.com","password":"wrong"}`, 401, "invalid_credentials"},
		{"login", "{", 400, "invalid_request"},
	} {
		status, body = post(tt.path, tt.body)
		failure, ok := body["error"].(map[string]any)
		if status != tt.status || !ok || failure["code"] != tt.code {
			t.Fatalf("%s = %d %#v", tt.path, status, body)
		}
	}
	db.Close()
	for _, path := range []string{"register", "login"} {
		status, body = post(path, credentials)
		failure, ok := body["error"].(map[string]any)
		if status != 500 || !ok || failure["code"] != "internal_error" || failure["message"] != "An unexpected error occurred." {
			t.Fatalf("DB failure for %s = %d %#v", path, status, body)
		}
	}
}
