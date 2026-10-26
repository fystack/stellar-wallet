package handler

import (
	"net/http"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/httpapi"
	"stellar-wallet-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type SwapHandler struct {
	swaps *service.SwapService
}

func (h *SwapHandler) quote(c *gin.Context) {
	input := service.SwapInput{From: c.Query("from"), To: c.Query("to"), Amount: c.Query("amount")}
	quote, err := h.swaps.Quote(c.GetString("userID"), c.Param("id"), input)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusOK, quote)
}

func (h *SwapHandler) create(c *gin.Context) {
	var input service.SwapInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpapi.Error(c, apperror.InvalidRequest.New())
		return
	}
	result, err := h.swaps.Create(c.GetString("userID"), c.Param("id"), input)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.Success(c, http.StatusCreated, result)
}
