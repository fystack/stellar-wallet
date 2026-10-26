package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func TestGoogleValidatesSignatureAudienceExpiryNonceAndEmail(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"valid", "wrong audience", "expired", "wrong nonce", "unverified email", "bad signature", "wrong issuer"} {
		t.Run(scenario, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": "https://accounts.google.com", "aud": "client-id", "sub": "google-user", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "expected-nonce", "email": "Owner@example.com", "email_verified": true}
			signingKey := key
			switch scenario {
			case "wrong audience":
				claims["aud"] = "another-client"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "wrong nonce":
				claims["nonce"] = "another-flow"
			case "unverified email":
				claims["email_verified"] = false
			case "bad signature":
				signingKey = wrongKey
			case "wrong issuer":
				claims["iss"] = "https://evil.test"
			}
			idToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(signingKey)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.Form.Get("code_verifier") != "expected-verifier" || r.Form.Get("code") != "auth-code" {
					t.Error("missing PKCE verifier or code")
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"access_token": "provider-token", "token_type": "Bearer", "id_token": idToken})
			}))
			defer server.Close()
			p := &provider{name: "google", client: server.Client(), oauth: oauth2.Config{ClientID: "client-id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{AuthURL: server.URL + "/authorize", TokenURL: server.URL, AuthStyle: oauth2.AuthStyleInParams}}, verifier: oidc.NewVerifier("https://accounts.google.com", &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "client-id"})}
			identity, err := p.Identity(context.Background(), "auth-code", "expected-verifier", "expected-nonce")
			if scenario == "valid" {
				if err != nil || identity.Subject != "google-user" || identity.Email != "owner@example.com" {
					t.Fatalf("identity=%#v %v", identity, err)
				}
			} else if err == nil {
				t.Fatalf("accepted invalid token: %s", scenario)
			}
			u, _ := url.Parse(p.AuthorizationURL("state", "expected-verifier", "nonce"))
			if u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("code_challenge") != oauth2.S256ChallengeFromVerifier("expected-verifier") || u.Query().Get("nonce") != "nonce" {
				t.Fatalf("authorization URL missing PKCE/nonce: %s", u)
			}
		})
	}
}

func TestGitHubUsesStableIDAndVerifiedEmail(t *testing.T) {
	for _, verified := range []bool{true, false} {
		t.Run(map[bool]string{true: "verified", false: "unverified"}[verified], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					r.ParseForm()
					if r.Form.Get("code_verifier") != "verifier" {
						t.Error("missing PKCE")
					}
					json.NewEncoder(w).Encode(map[string]string{"access_token": "token", "token_type": "Bearer"})
					return
				}
				if r.Header.Get("Authorization") != "Bearer token" {
					t.Error("missing provider authorization")
				}
				if r.URL.Path == "/user" {
					json.NewEncoder(w).Encode(map[string]any{"id": 12345, "email": "untrusted-public@example.com"})
					return
				}
				json.NewEncoder(w).Encode([]map[string]any{{"email": "unverified@example.com", "primary": true, "verified": false}, {"email": "verified@example.com", "primary": false, "verified": verified}})
			}))
			defer server.Close()
			p := &provider{name: "github", client: server.Client(), userURL: server.URL + "/user", emailsURL: server.URL + "/emails", oauth: oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: server.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}}
			identity, err := p.Identity(context.Background(), "code", "verifier", "")
			if verified {
				if err != nil || identity.Subject != "12345" || identity.Email != "verified@example.com" {
					t.Fatalf("identity=%#v %v", identity, err)
				}
			} else if err == nil {
				t.Fatal("unverified email accepted")
			}
		})
	}
}
