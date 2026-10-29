package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	sessions    *auth.Manager
	accounts    *service.AuthService
	frontendURL string
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) register(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	account, token, err := h.accounts.Register(body.Email, body.Password, auth.CookieToken(c.Request))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.SetSession(c, token)
	httpapi.Success(c, http.StatusCreated, account)
}
func (h *AuthHandler) login(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	account, token, err := h.accounts.Login(body.Email, body.Password, auth.CookieToken(c.Request))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.SetSession(c, token)
	httpapi.Success(c, http.StatusOK, account)
}
func (h *AuthHandler) me(c *gin.Context) {
	account, err := h.accounts.Account(c.GetString("userID"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, account)
}
func (h *AuthHandler) logout(c *gin.Context) {
	if err := h.accounts.Logout(auth.CookieToken(c.Request)); err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.ClearSession(c)
	httpapi.Success(c, http.StatusNoContent, nil)
}
func (h *AuthHandler) logoutEverywhere(c *gin.Context) {
	if err := h.accounts.LogoutEverywhere(c.GetString("userID")); err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.ClearSession(c)
	httpapi.Success(c, http.StatusNoContent, nil)
}
func (h *AuthHandler) reauthenticate(c *gin.Context) {
	var body credentials
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	token, err := h.accounts.Reauthenticate(c.GetString("userID"), body.Password, auth.CookieToken(c.Request))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.SetSession(c, token)
	httpapi.Success(c, http.StatusNoContent, nil)
}
func (h *AuthHandler) providers(c *gin.Context) {
	httpapi.Success(c, http.StatusOK, h.accounts.Providers())
}
func (h *AuthHandler) oauthStart(c *gin.Context) {
	start, err := h.accounts.StartOAuth(c.Param("provider"), auth.CookieToken(c.Request), false)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.SetCookie(c, auth.OAuthCookie, start.BrowserToken, 600)
	c.Redirect(http.StatusFound, start.URL)
}
func (h *AuthHandler) oauthLink(c *gin.Context) {
	start, err := h.accounts.StartOAuth(c.Param("provider"), auth.CookieToken(c.Request), true)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	h.sessions.SetCookie(c, auth.OAuthCookie, start.BrowserToken, 600)
	httpapi.Success(c, http.StatusOK, gin.H{"url": start.URL})
}
func (h *AuthHandler) oauthCallback(c *gin.Context) {
	browser, _ := c.Cookie(auth.OAuthCookie)
	h.sessions.SetCookie(c, auth.OAuthCookie, "", -1)
	result, err := h.accounts.CompleteOAuth(c.Request.Context(), c.Param("provider"), c.Query("state"), browser, c.Query("code"), auth.CookieToken(c.Request), c.Query("error") != "")
	destination := strings.TrimRight(h.frontendURL, "/")
	fragment := "#/wallets"
	if result.Linked {
		fragment = "#/settings"
	}
	if err != nil {
		code := apperror.OAuthFailed.Code()
		var failure *apperror.Error
		if errors.As(err, &failure) {
			code = failure.Code
		}
		// Never log authorization codes, state, cookies, tokens or raw provider bodies.
		zap.L().Warn("OAuth callback failed", zap.String("provider", c.Param("provider")), zap.String("code", code))
		destination += "?auth_error=" + url.QueryEscape(code)
	} else if result.Token != "" {
		h.sessions.SetSession(c, result.Token)
	}
	c.Redirect(http.StatusFound, destination+fragment)
}
