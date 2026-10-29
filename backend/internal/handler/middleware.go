package handler

import (
	"fmt"
	"net/http"
	"strings"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"

	"github.com/gin-gonic/gin"
)

// requestLogger logs the path only: OAuth callback queries contain single-use secrets.
func requestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(p gin.LogFormatterParams) string {
		return fmt.Sprintf("%s %s %d %s\n", p.Method, p.Request.URL.Path, p.StatusCode, p.Latency)
	})
}

func recoverPanics() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				httpapi.Error(c, fmt.Errorf("request handler panicked (%T)", recovered))
			}
		}()
		c.Next()
	}
}

func corsMiddleware(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Protection")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// csrfProtection requires the custom header on unsafe methods and rejects foreign origins.
func csrfProtection(frontendURL, publicURL string) gin.HandlerFunc {
	frontendURL = strings.TrimRight(frontendURL, "/")
	publicURL = strings.TrimRight(publicURL, "/")
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if c.GetHeader("X-CSRF-Protection") != "1" || (origin != "" && origin != frontendURL && origin != publicURL) {
			httpapi.Error(c, apperror.CSRFFailed.New())
			return
		}
		c.Next()
	}
}
