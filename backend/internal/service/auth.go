package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"
	"stellar-wallet-backend/internal/repository/sqlite"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

type AuthService struct {
	repo      *sqlite.AuthRepository
	providers map[string]auth.OAuthProvider
}

func NewAuthService(repo *sqlite.AuthRepository, providers map[string]auth.OAuthProvider) *AuthService {
	return &AuthService{repo: repo, providers: providers}
}

type SocialProvider struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

func (s *AuthService) Providers() []SocialProvider {
	return []SocialProvider{{ID: "google", Enabled: s.providers["google"] != nil}, {ID: "github", Enabled: s.providers["github"] != nil}}
}

func (s *AuthService) Register(email, password, oldToken string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || utf8.RuneCountInString(password) < 6 || len([]byte(password)) > 72 {
		return domain.User{}, "", apperror.InvalidRegistration.New()
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}
	user := domain.User{ID: uuid.NewString(), Email: email, PasswordHash: string(hash), CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := s.repo.CreateUser(user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return user, "", apperror.EmailAlreadyRegistered.New()
		}
		return user, "", fmt.Errorf("register user: %w", err)
	}
	token, err := s.newSession(user.ID, oldToken)
	return user, token, err
}
func (s *AuthService) Login(email, password, oldToken string) (domain.User, string, error) {
	id, hash, err := s.repo.UserCredentials(strings.ToLower(strings.TrimSpace(email)))
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.User{}, "", fmt.Errorf("read login credentials: %w", err)
	}
	if hash == "" {
		// Compare against a dummy hash so unknown emails take as long as wrong passwords.
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return domain.User{}, "", apperror.InvalidLoginCredentials.New()
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return domain.User{}, "", apperror.InvalidLoginCredentials.New()
	}
	user, err := s.repo.UserByID(id)
	if err != nil {
		return user, "", err
	}
	token, err := s.newSession(user.ID, oldToken)
	return user, token, err
}
func (s *AuthService) newSession(userID, oldToken string) (string, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	now := time.Now()
	if err := s.repo.CreateSession(domain.Session{Hash: auth.HashToken(token), UserID: userID, CreatedAt: now.Unix(), ExpiresAt: now.Add(auth.SessionTTL).Unix()}); err != nil {
		return "", err
	}
	if oldToken != "" {
		if err := s.Logout(oldToken); err != nil {
			s.repo.RevokeSession(auth.HashToken(token))
			return "", err
		}
	}
	return token, nil
}
func (s *AuthService) Logout(token string) error                     { return s.repo.RevokeSession(auth.HashToken(token)) }
func (s *AuthService) LogoutEverywhere(userID string) error          { return s.repo.RevokeUserSessions(userID) }
func (s *AuthService) Account(userID string) (domain.Account, error) { return s.repo.Account(userID) }
func (s *AuthService) Reauthenticate(userID, password, oldToken string) (string, error) {
	session, err := s.repo.Session(auth.HashToken(oldToken))
	if errors.Is(err, repository.ErrNotFound) {
		return "", apperror.AuthenticationRequired.New()
	}
	if err != nil {
		return "", err
	}
	if session.UserID != userID {
		return "", apperror.AuthenticationRequired.New()
	}
	user, err := s.repo.UserByID(userID)
	if err != nil {
		return "", err
	}
	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", apperror.InvalidPassword.New()
	}
	return s.newSession(userID, oldToken)
}

type OAuthStart struct {
	URL          string
	BrowserToken string
}
type OAuthResult struct {
	User   domain.User
	Token  string
	Linked bool
}

func (s *AuthService) provider(name string) (auth.OAuthProvider, error) {
	if name != "google" && name != "github" {
		return nil, apperror.UnsupportedProvider.New()
	}
	provider := s.providers[name]
	if provider == nil {
		return nil, apperror.ProviderUnavailable.New()
	}
	return provider, nil
}
func (s *AuthService) recentSession(token string) (domain.Session, error) {
	session, err := s.repo.Session(auth.HashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return session, apperror.AuthenticationRequired.New()
	}
	if err != nil {
		return session, err
	}
	if time.Now().Unix()-session.CreatedAt > int64((10 * time.Minute).Seconds()) {
		return session, apperror.ReauthenticationRequired.New()
	}
	return session, nil
}
func (s *AuthService) StartOAuth(name, sessionToken string, link bool) (OAuthStart, error) {
	provider, err := s.provider(name)
	if err != nil {
		return OAuthStart{}, err
	}
	state, err := auth.RandomToken()
	if err != nil {
		return OAuthStart{}, fmt.Errorf("generate OAuth state: %w", err)
	}
	browser, err := auth.RandomToken()
	if err != nil {
		return OAuthStart{}, fmt.Errorf("generate OAuth browser token: %w", err)
	}
	nonce, err := auth.RandomToken()
	if err != nil {
		return OAuthStart{}, fmt.Errorf("generate OAuth nonce: %w", err)
	}
	flow := domain.OAuthFlow{StateHash: auth.HashToken(state), BrowserHash: auth.HashToken(browser), Provider: name, Verifier: oauth2.GenerateVerifier(), Nonce: nonce, ExpiresAt: time.Now().Add(10 * time.Minute).Unix()}
	if link {
		session, err := s.recentSession(sessionToken)
		if err != nil {
			return OAuthStart{}, err
		}
		flow.UserID, flow.SessionHash = session.UserID, session.Hash
	}
	if err := s.repo.CreateFlow(flow); err != nil {
		return OAuthStart{}, err
	}
	return OAuthStart{URL: provider.AuthorizationURL(state, flow.Verifier, nonce), BrowserToken: browser}, nil
}
func (s *AuthService) CompleteOAuth(ctx context.Context, name, state, browser, code, sessionToken string, denied bool) (OAuthResult, error) {
	result := OAuthResult{}
	provider, err := s.provider(name)
	if err != nil {
		return result, err
	}
	if state == "" || browser == "" {
		return result, apperror.InvalidOAuthState.New()
	}
	flow, err := s.repo.ConsumeFlow(auth.HashToken(state), auth.HashToken(browser), name)
	if errors.Is(err, repository.ErrNotFound) {
		return result, apperror.InvalidOAuthState.New()
	}
	if err != nil {
		return result, err
	}
	result.Linked = flow.UserID != ""
	if result.Linked {
		session, err := s.recentSession(sessionToken)
		if err != nil {
			return result, err
		}
		if session.Hash != flow.SessionHash || session.UserID != flow.UserID {
			return result, apperror.InvalidLinkSession.New()
		}
	}
	if denied || code == "" {
		return result, apperror.OAuthCancelled.New()
	}
	identity, err := provider.Identity(ctx, code, flow.Verifier, flow.Nonce)
	if err != nil {
		return result, apperror.OAuthFailed.Wrap(err)
	}
	if identity.Provider != name || identity.Subject == "" || identity.Email == "" {
		return result, apperror.OAuthFailed.New()
	}
	user, err := s.repo.SocialUser(identity, flow.UserID)
	if errors.Is(err, repository.ErrConflict) {
		return result, apperror.AccountLinkRequired.New()
	}
	if errors.Is(err, repository.ErrIdentityLinked) {
		return result, apperror.IdentityAlreadyLinked.New()
	}
	if err != nil {
		return result, fmt.Errorf("complete social login: %w", err)
	}
	result.User = user
	if !result.Linked {
		result.Token, err = s.newSession(user.ID, sessionToken)
	}
	return result, err
}
