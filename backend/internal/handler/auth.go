package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Handler) register(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Email) == "" || len(body.Password) < 6 {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_request", "Email and password (minimum 6 characters) are required."))
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		httpapi.Error(c, fmt.Errorf("hash registration password: %w", err))
		return
	}
	userID := uuid.NewString()
	err = s.store.CreateUser(domain.User{ID: userID, Email: email, PasswordHash: string(hash), CreatedAt: time.Now().UTC().Format(time.RFC3339)})
	if errors.Is(err, repository.ErrConflict) {
		httpapi.Error(c, apperror.Wrap(err, apperror.Conflict, "email_already_registered", "Email is already registered."))
		return
	}
	if err != nil {
		httpapi.Error(c, fmt.Errorf("register user: %w", err))
		return
	}
	token, ttl, err := s.auth.MakeToken(userID)
	if err != nil {
		httpapi.Error(c, fmt.Errorf("issue registration token: %w", err))
		return
	}
	httpapi.Success(c, http.StatusCreated, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}

func (s *Handler) login(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.New(apperror.Invalid, "invalid_request", "Invalid request body."))
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	userID, hash, err := s.store.UserCredentials(email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		httpapi.Error(c, fmt.Errorf("read login credentials: %w", err))
		return
	}
	if errors.Is(err, repository.ErrNotFound) || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		httpapi.Error(c, apperror.New(apperror.Unauthorized, "invalid_credentials", "Invalid email or password."))
		return
	}
	token, ttl, err := s.auth.MakeToken(userID)
	if err != nil {
		httpapi.Error(c, fmt.Errorf("issue login token: %w", err))
		return
	}
	httpapi.Success(c, http.StatusOK, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}
