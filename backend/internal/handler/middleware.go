package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// requestLogger logs the route pattern only: OAuth callback queries and invite paths contain single-use secrets.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		zap.L().Info("request",
			zap.String("method", c.Request.Method),
			zap.String("route", route),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
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
