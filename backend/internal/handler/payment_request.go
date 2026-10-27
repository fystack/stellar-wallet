package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type PaymentRequestHandler struct {
	requests *service.PaymentRequestService
}

func (h *PaymentRequestHandler) list(c *gin.Context) {
	requests, err := h.requests.List(c.GetString("userID"), c.Param("id"), c.Query("status"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, requests)
}

func (h *PaymentRequestHandler) get(c *gin.Context) {
	request, err := h.requests.Get(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, request)
}

func (h *PaymentRequestHandler) approve(c *gin.Context) {
	request, err := h.requests.Approve(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, request)
}

func (h *PaymentRequestHandler) reject(c *gin.Context) {
	request, err := h.requests.Reject(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, request)
}

func (h *PaymentRequestHandler) cancel(c *gin.Context) {
	request, err := h.requests.Cancel(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, request)
}
