package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *server) listWalletTxns(c *gin.Context) {
	userID := c.GetString("userID")
	walletID := c.Param("id")
	if _, ok := s.walletFor(userID, walletID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, s.txnsWhere(`wallet_id = ? AND user_id = ?`, walletID, userID))
}

func (s *server) getTxn(c *gin.Context) {
	txns := s.txnsWhere(`id = ? AND user_id = ?`, c.Param("id"), c.GetString("userID"))
	if len(txns) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	c.JSON(http.StatusOK, txns[0])
}

type sendBody struct {
	WalletID    string `json:"wallet_id"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Memo        string `json:"memo"`
	AssetCode   string `json:"asset_code"`   // empty / "XLM" = native
	AssetIssuer string `json:"asset_issuer"` // required for non-native
}

// createTxn records an outgoing transfer and asks the MPC cluster to sign it.
// The signature arrives asynchronously via the sign result (pushed over SSE).
func (s *server) createTxn(c *gin.Context) {
	userID := c.GetString("userID")
	var body sendBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	w, ok := s.walletFor(userID, body.WalletID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	if w.Status != "ready" {
		c.JSON(http.StatusConflict, gin.H{"error": "wallet is not ready yet"})
		return
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(body.Amount), 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
		return
	}
	if strings.TrimSpace(body.To) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient required"})
		return
	}

	symbol := body.AssetCode
	if symbol == "" {
		symbol = w.Symbol
	}
	tx := Transaction{
		ID:           uuid.NewString(),
		WalletID:     w.ID,
		UserID:       userID,
		Type:         "out",
		Counterparty: strings.TrimSpace(body.To),
		Amount:       strconv.FormatFloat(amount, 'f', 2, 64),
		Symbol:       symbol,
		Memo:         strings.TrimSpace(body.Memo),
		Status:       "signing",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	// Build the payload the cluster will sign.
	var toSign []byte
	if w.Chain == "stellar" {
		env, hash, err := stellarBuildPayment(
			w, tx.Counterparty, tx.Amount, body.AssetCode, body.AssetIssuer, tx.Memo)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		tx.EnvelopeXDR = env
		toSign = hash
	} else {
		// Solana: no on-chain broadcast yet — sign a deterministic payload.
		toSign = []byte(fmt.Sprintf("%s|%s|%s|%s", w.ID, tx.Counterparty, tx.Amount, tx.Memo))
	}

	_, err = s.db.Exec(
		`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, memo, status, signature, envelope_xdr, tx_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?)`,
		tx.ID, tx.WalletID, tx.UserID, tx.Type, tx.Counterparty, tx.Amount, tx.Symbol, tx.Memo, tx.Status, tx.EnvelopeXDR, tx.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}

	if err := s.mpc.Sign(w.ID, tx.ID, chainMeta[w.Chain].Network, toSign); err != nil {
		s.db.Exec(`UPDATE transactions SET status = 'failed' WHERE id = ?`, tx.ID)
		c.JSON(http.StatusBadGateway, gin.H{"error": "sign dispatch failed: " + err.Error()})
		return
	}

	s.hub.publishTxn(userID, tx)
	c.JSON(http.StatusOK, tx)
}

type trustlineBody struct {
	Code   string `json:"code"`
	Issuer string `json:"issuer"`
}

// addTrustline lets a Stellar wallet hold a custom asset (changeTrust op),
// signed by the cluster and broadcast — reuses the normal sign/broadcast path.
func (s *server) addTrustline(c *gin.Context) {
	userID := c.GetString("userID")
	w, ok := s.walletFor(userID, c.Param("id"))
	if !ok || w.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	if w.Chain != "stellar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trustlines are Stellar-only"})
		return
	}
	var body trustlineBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Code == "" || body.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and issuer required"})
		return
	}

	// Guard against duplicate trustlines (double-click / concurrent requests).
	var dup int
	s.db.QueryRow(
		`SELECT COUNT(1) FROM transactions
		 WHERE wallet_id = ? AND symbol = ? AND memo = 'Add trustline'
		   AND status IN ('signing','broadcast','confirmed')`, w.ID, body.Code).Scan(&dup)
	if dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "trustline already added or in progress"})
		return
	}

	env, hash, err := stellarBuildTrustline(w, body.Code, body.Issuer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tx := Transaction{
		ID:           uuid.NewString(),
		WalletID:     w.ID,
		UserID:       userID,
		Type:         "out",
		Counterparty: body.Issuer,
		Amount:       "0.00",
		Symbol:       body.Code,
		Memo:         "Add trustline",
		Status:       "signing",
		EnvelopeXDR:  env,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	_, err = s.db.Exec(
		`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, memo, status, signature, envelope_xdr, tx_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, '', ?)`,
		tx.ID, tx.WalletID, tx.UserID, tx.Type, tx.Counterparty, tx.Amount, tx.Symbol, tx.Memo, tx.Status, tx.EnvelopeXDR, tx.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	if err := s.mpc.Sign(w.ID, tx.ID, chainMeta[w.Chain].Network, hash); err != nil {
		s.db.Exec(`UPDATE transactions SET status = 'failed' WHERE id = ?`, tx.ID)
		c.JSON(http.StatusBadGateway, gin.H{"error": "sign dispatch failed: " + err.Error()})
		return
	}
	s.hub.publishTxn(userID, tx)
	c.JSON(http.StatusOK, tx)
}

func (s *server) txnsWhere(where string, args ...any) []Transaction {
	rows, err := s.db.Query(
		`SELECT id, wallet_id, type, counterparty, amount, symbol, COALESCE(memo, ''), status, signature, envelope_xdr, tx_hash, error, created_at
		 FROM transactions WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return []Transaction{}
	}
	defer rows.Close()

	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.WalletID, &t.Type, &t.Counterparty, &t.Amount, &t.Symbol, &t.Memo, &t.Status, &t.Signature, &t.EnvelopeXDR, &t.TxHash, &t.Error, &t.CreatedAt); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out
}
