package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"stellar-wallet-backend/internal/config"
	"stellar-wallet-backend/internal/domain"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OAuthProvider is the external identity boundary used by AuthService.
type OAuthProvider interface {
	AuthorizationURL(state, verifier, nonce string) string
	Identity(context.Context, string, string, string) (domain.Identity, error)
}

type provider struct {
	name      string
	oauth     oauth2.Config
	client    *http.Client
	verifier  *oidc.IDTokenVerifier
	userURL   string
	emailsURL string
}

// NewProviders only enables providers with configured client credentials.
func NewProviders(cfg config.OAuthConfig, publicURL string, client *http.Client) map[string]OAuthProvider {
	providers := map[string]OAuthProvider{}
	for _, name := range []string{"google", "github"} {
		credentials := cfg.Google
		if name == "github" {
			credentials = cfg.GitHub
		}
		if credentials.ClientID == "" || credentials.ClientSecret == "" {
			continue
		}
		p := &provider{name: name, client: client, oauth: oauth2.Config{ClientID: credentials.ClientID, ClientSecret: credentials.ClientSecret, RedirectURL: strings.TrimRight(publicURL, "/") + "/api/v1/auth/" + name + "/callback"}}
		p.oauth.Endpoint.AuthStyle = oauth2.AuthStyleInParams
		if name == "google" {
			p.oauth.Endpoint.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
			p.oauth.Endpoint.TokenURL = "https://oauth2.googleapis.com/token"
			p.oauth.Scopes = []string{"openid", "email", "profile"}
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
			keys := oidc.NewRemoteKeySet(ctx, "https://www.googleapis.com/oauth2/v3/certs")
			p.verifier = oidc.NewVerifier("https://accounts.google.com", keys, &oidc.Config{ClientID: credentials.ClientID})
		} else {
			p.oauth.Endpoint.AuthURL = "https://github.com/login/oauth/authorize"
			p.oauth.Endpoint.TokenURL = "https://github.com/login/oauth/access_token"
			p.oauth.Scopes = []string{"read:user", "user:email"}
			p.userURL = "https://api.github.com/user"
			p.emailsURL = "https://api.github.com/user/emails?per_page=100"
		}
		providers[name] = p
	}
	return providers
}

func (p *provider) AuthorizationURL(state, verifier, nonce string) string {
	options := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier)}
	if p.name == "google" {
		options = append(options, oauth2.SetAuthURLParam("nonce", nonce))
	}
	return p.oauth.AuthCodeURL(state, options...)
}

func (p *provider) Identity(ctx context.Context, code, verifier, nonce string) (domain.Identity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := p.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return domain.Identity{}, fmt.Errorf("exchange %s authorization code: %w", p.name, err)
	}
	if p.name == "google" {
		// The nonce binds the ID token to this login.
		raw, ok := token.Extra("id_token").(string)
		if !ok {
			return domain.Identity{}, fmt.Errorf("Google response missing ID token")
		}
		idToken, err := p.verifier.Verify(ctx, raw)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("verify Google ID token: %w", err)
		}
		if idToken.Nonce != nonce || nonce == "" {
			return domain.Identity{}, fmt.Errorf("Google nonce mismatch")
		}
		var claims struct {
			Email    string `json:"email"`
			Verified bool   `json:"email_verified"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return domain.Identity{}, fmt.Errorf("decode Google claims: %w", err)
		}
		if idToken.Subject == "" || !claims.Verified || strings.TrimSpace(claims.Email) == "" {
			return domain.Identity{}, fmt.Errorf("Google identity requires a verified email")
		}
		return domain.Identity{Provider: p.name, Subject: idToken.Subject, Email: strings.ToLower(strings.TrimSpace(claims.Email))}, nil
	}
	var user struct {
		ID json.Number `json:"id"`
	}
	if err := p.getJSON(ctx, p.userURL, token.AccessToken, &user); err != nil {
		return domain.Identity{}, err
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := p.getJSON(ctx, p.emailsURL, token.AccessToken, &emails); err != nil {
		return domain.Identity{}, err
	}
	email := ""
	for _, entry := range emails {
		if entry.Verified {
			if email == "" || entry.Primary {
				email = entry.Email
			}
			if entry.Primary {
				break
			}
		}
	}
	if user.ID == "" || user.ID == "0" || strings.TrimSpace(email) == "" {
		return domain.Identity{}, fmt.Errorf("GitHub identity requires a user ID and verified email")
	}
	return domain.Identity{Provider: p.name, Subject: user.ID.String(), Email: strings.ToLower(strings.TrimSpace(email))}, nil
}

func (p *provider) getJSON(ctx context.Context, endpoint, token string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	response, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("read provider profile: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("provider profile returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode provider profile: %w", err)
	}
	return nil
}
