// Package httpapi owns the JSON response contract and HTTP error mapping.
package httpapi

import (
	"errors"
	"net/http"

	"stellar-wallet-backend/internal/apperror"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Response struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type CursorMeta struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Cursor builds the pagination meta; an empty next means the last page.
func Cursor(next string) CursorMeta {
	if next == "" {
		return CursorMeta{}
	}
	return CursorMeta{NextCursor: &next, HasMore: true}
}

func Success(c *gin.Context, status int, data any) {
	SuccessWithMeta(c, status, data, nil)
}

func SuccessWithMeta(c *gin.Context, status int, data, meta any) {
	if status == http.StatusNoContent {
		c.Status(status)
		return
	}
	c.JSON(status, Response{Data: data, Meta: meta})
}

// Error aborts the request. Only explicitly public messages reach the client.
func Error(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	internal := apperror.InternalFailure.New()
	detail := ErrorDetail{Code: internal.Code, Message: internal.Message}
	var failure *apperror.Error
	if errors.As(err, &failure) && failure.Kind != apperror.Internal {
		switch failure.Kind {
		case apperror.Invalid:
			status = http.StatusBadRequest
		case apperror.Unauthorized:
			status = http.StatusUnauthorized
		case apperror.Forbidden:
			status = http.StatusForbidden
		case apperror.NotFound:
			status = http.StatusNotFound
		case apperror.Conflict:
			status = http.StatusConflict
		case apperror.Unavailable:
			status = http.StatusServiceUnavailable
		case apperror.Upstream:
			status = http.StatusBadGateway
		case apperror.MethodNotAllowed:
			status = http.StatusMethodNotAllowed
		case apperror.TooManyRequests:
			status = http.StatusTooManyRequests
		}
		if status != http.StatusInternalServerError {
			detail = ErrorDetail{Code: failure.Code, Message: failure.Message}
		}
	}
	if status >= 500 || (failure != nil && failure.Unwrap() != nil) {
		zap.L().Error("request failed", zap.String("method", c.Request.Method), zap.String("route", c.FullPath()), zap.Int("status", status), zap.Error(err))
	}
	c.AbortWithStatusJSON(status, ErrorResponse{Error: detail})
}
