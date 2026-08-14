package app

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) register(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || len(body.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password (min 6 chars) required"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	userID := uuid.NewString()
	err = s.store.CreateUser(domain.User{
		ID: userID, Email: email, PasswordHash: string(hash),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}
	token, ttl, _ := s.auth.MakeToken(userID)
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}

func (s *Server) login(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	userID, hash, err := s.store.UserCredentials(email)
	if errors.Is(err, store.ErrNotFound) || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	token, ttl, _ := s.auth.MakeToken(userID)
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}
