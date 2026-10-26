package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	modernsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
	"stellar-wallet-backend/internal/domain"
	storecontract "stellar-wallet-backend/internal/repository"
)

// AuthRepository owns account, identity, session and short-lived OAuth state SQL.
type AuthRepository struct{ db *sql.DB }

func NewAuthRepository(db *sql.DB) *AuthRepository { return &AuthRepository{db: db} }

func (s *AuthRepository) CreateUser(user domain.User) error {
	_, err := s.db.Exec(
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt,
	)
	if err != nil {
		var sqliteErr *modernsqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return fmt.Errorf("create user: %w: %w", storecontract.ErrConflict, err)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *AuthRepository) UserCredentials(email string) (string, string, error) {
	var userID, passwordHash string
	err := s.db.QueryRow(
		`SELECT id, password_hash FROM users WHERE email = ?`, email,
	).Scan(&userID, &passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("read user credentials: %w", storecontract.ErrNotFound)
	}
	if err != nil {
		return "", "", fmt.Errorf("read user credentials: %w", err)
	}
	return userID, passwordHash, nil
}

func (s *AuthRepository) UserByID(id string) (domain.User, error) {
	var u domain.User
	err := s.db.QueryRow(`SELECT id,email,password_hash,created_at FROM users WHERE id=?`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, storecontract.ErrNotFound
	}
	if err != nil {
		return u, fmt.Errorf("read user: %w", err)
	}
	return u, nil
}

func (s *AuthRepository) Account(userID string) (domain.Account, error) {
	u, err := s.UserByID(userID)
	if err != nil {
		return domain.Account{}, err
	}
	account := domain.Account{ID: u.ID, Email: u.Email, HasPassword: u.PasswordHash != "", Identities: []domain.Identity{}}
	rows, err := s.db.Query(`SELECT provider,subject,email FROM auth_identities WHERE user_id=? ORDER BY provider`, userID)
	if err != nil {
		return account, fmt.Errorf("list identities: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var i domain.Identity
		if err := rows.Scan(&i.Provider, &i.Subject, &i.Email); err != nil {
			return account, fmt.Errorf("scan identity: %w", err)
		}
		account.Identities = append(account.Identities, i)
	}
	return account, rows.Err()
}

// SocialUser resolves an identity, or links it atomically to an authenticated user.
// A matching email alone never authorizes account linking.
func (s *AuthRepository) SocialUser(identity domain.Identity, linkUserID string) (domain.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return domain.User{}, fmt.Errorf("begin social login: %w", err)
	}
	defer tx.Rollback()
	var userID string
	err = tx.QueryRow(`SELECT user_id FROM auth_identities WHERE provider=? AND subject=?`, identity.Provider, identity.Subject).Scan(&userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("find identity: %w", err)
	}
	if userID != "" && linkUserID != "" && userID != linkUserID {
		return domain.User{}, storecontract.ErrIdentityLinked
	}
	if userID == "" {
		userID = linkUserID
		if userID == "" {
			var exists int
			if err := tx.QueryRow(`SELECT count(*) FROM users WHERE email=?`, identity.Email).Scan(&exists); err != nil {
				return domain.User{}, err
			}
			if exists > 0 {
				return domain.User{}, storecontract.ErrConflict
			}
			userID = uuid.NewString()
			if _, err := tx.Exec(`INSERT INTO users(id,email,password_hash,created_at) VALUES(?,?,?,?)`, userID, identity.Email, "", time.Now().UTC().Format(time.RFC3339)); err != nil {
				return domain.User{}, fmt.Errorf("create social user: %w", err)
			}
		}
		var exists int
		if err := tx.QueryRow(`SELECT count(*) FROM auth_identities WHERE user_id=? AND provider=?`, userID, identity.Provider).Scan(&exists); err != nil {
			return domain.User{}, err
		}
		if exists > 0 {
			return domain.User{}, storecontract.ErrIdentityLinked
		}
		if _, err := tx.Exec(`INSERT INTO auth_identities(provider,subject,user_id,email) VALUES(?,?,?,?)`, identity.Provider, identity.Subject, userID, identity.Email); err != nil {
			return domain.User{}, fmt.Errorf("link identity: %w", err)
		}
	}
	var u domain.User
	if err := tx.QueryRow(`SELECT id,email,password_hash,created_at FROM users WHERE id=?`, userID).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return u, fmt.Errorf("read social user: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return u, fmt.Errorf("commit social login: %w", err)
	}
	return u, nil
}

func (s *AuthRepository) CreateSession(session domain.Session) error {
	if _, err := s.db.Exec(`DELETE FROM auth_sessions WHERE expires_at<=?`, time.Now().Unix()); err != nil {
		return fmt.Errorf("clean expired sessions: %w", err)
	}
	_, err := s.db.Exec(`INSERT INTO auth_sessions(token_hash,user_id,created_at,expires_at) VALUES(?,?,?,?)`, session.Hash, session.UserID, session.CreatedAt, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}
func (s *AuthRepository) Session(hash string) (domain.Session, error) {
	var session domain.Session
	err := s.db.QueryRow(`SELECT token_hash,user_id,created_at,expires_at FROM auth_sessions WHERE token_hash=? AND expires_at>?`, hash, time.Now().Unix()).Scan(&session.Hash, &session.UserID, &session.CreatedAt, &session.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return session, storecontract.ErrNotFound
	}
	if err != nil {
		return session, fmt.Errorf("read session: %w", err)
	}
	return session, nil
}
func (s *AuthRepository) RevokeSession(hash string) error {
	_, err := s.db.Exec(`DELETE FROM auth_sessions WHERE token_hash=?`, hash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
func (s *AuthRepository) CreateFlow(flow domain.OAuthFlow) error {
	if _, err := s.db.Exec(`DELETE FROM oauth_flows WHERE expires_at<=?`, time.Now().Unix()); err != nil {
		return fmt.Errorf("clean expired OAuth states: %w", err)
	}
	_, err := s.db.Exec(`INSERT INTO oauth_flows(state_hash,browser_hash,provider,verifier,nonce,session_hash,user_id,expires_at) VALUES(?,?,?,?,?,?,?,?)`, flow.StateHash, flow.BrowserHash, flow.Provider, flow.Verifier, flow.Nonce, flow.SessionHash, flow.UserID, flow.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create OAuth state: %w", err)
	}
	return nil
}
func (s *AuthRepository) ConsumeFlow(stateHash, browserHash, provider string) (domain.OAuthFlow, error) {
	var f domain.OAuthFlow
	err := s.db.QueryRow(`DELETE FROM oauth_flows WHERE state_hash=? AND browser_hash=? AND provider=? AND expires_at>? RETURNING state_hash,browser_hash,provider,verifier,nonce,session_hash,user_id,expires_at`, stateHash, browserHash, provider, time.Now().Unix()).Scan(&f.StateHash, &f.BrowserHash, &f.Provider, &f.Verifier, &f.Nonce, &f.SessionHash, &f.UserID, &f.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return f, storecontract.ErrNotFound
	}
	if err != nil {
		return f, fmt.Errorf("consume OAuth state: %w", err)
	}
	return f, nil
}
