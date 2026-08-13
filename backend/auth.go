package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("dev-secret-change-me")

const tokenTTL = 24 * time.Hour

func makeToken(userID string) (string, int, error) {
	exp := time.Now().Add(tokenTTL)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": exp.Unix(),
	})
	s, err := tok.SignedString(jwtSecret)
	return s, int(tokenTTL.Seconds()), err
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *server) register(c *gin.Context) {
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

	id := uuid.NewString()
	_, err = s.db.Exec(
		`INSERT INTO users (id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, email, string(hash), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	token, ttl, _ := makeToken(id)
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}

func (s *server) login(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	var id, hash string
	err := s.db.QueryRow(`SELECT id, password_hash FROM users WHERE email = ?`, email).Scan(&id, &hash)
	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	token, ttl, _ := makeToken(id)
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_in": ttl, "email": email})
}

// parseToken validates a JWT string and returns the user id.
func parseToken(tokenStr string) (string, bool) {
	if tokenStr == "" {
		return "", false
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return "", false
	}
	sub, _ := claims["sub"].(string)
	return sub, sub != ""
}

// authMiddleware validates the Bearer token and stores the user id in the context.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		sub, ok := parseToken(parts[1])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("userID", sub)
		c.Next()
	}
}
