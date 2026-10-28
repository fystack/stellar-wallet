package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactions *service.TransactionService
}

func (h *TransactionHandler) listForWallet(c *gin.Context) {
	var page service.PageInput
	if err := c.ShouldBindQuery(&page); err != nil {
		httpapi.Error(c, apperror.InvalidPage.New())
		return
	}
	transactions, err := h.transactions.ListForWallet(c.GetString("userID"), c.Param("id"), page)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.SuccessWithMeta(c, http.StatusOK, transactions.Items, httpapi.Cursor(transactions.Next))
}

func (h *TransactionHandler) get(c *gin.Context) {
	transaction, err := h.transactions.Get(c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, transaction)
}

func (h *TransactionHandler) onChain(c *gin.Context) {
	metadata, err := h.transactions.OnChain(c.Param("hash"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, metadata)
}

func (h *TransactionHandler) resolveRecipient(c *gin.Context) {
	resolved, err := h.transactions.ResolveRecipient(c.Query("q"))
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, resolved)
}

func (h *TransactionHandler) create(c *gin.Context) {
	var body service.PaymentInput
	if err := c.ShouldBindJSON(&body); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	result, err := h.transactions.Create(c.GetString("userID"), body)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	status := http.StatusCreated
	if result.Request != nil {
		status = http.StatusAccepted
	}
	httpapi.Success(c, status, result)
}
