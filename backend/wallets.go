package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// syncIncoming pulls recent on-chain payments received by the wallet and
// records any that aren't in our DB yet (dedup by tx hash). Returns how many
// new incoming transactions were added.
func (s *server) syncIncoming(w Wallet) int {
	if w.Chain != "stellar" {
		return 0 // on-chain history sync implemented for Stellar for now
	}
	payments, err := stellarIncoming(w.Address)
	if err != nil {
		return 0
	}
	added := 0
	for _, p := range payments {
		var exists int
		s.db.QueryRow(
			`SELECT COUNT(1) FROM transactions WHERE tx_hash = ? AND type = 'in'`, p.Hash).
			Scan(&exists)
		if exists > 0 {
			continue
		}
		amt, _ := strconv.ParseFloat(p.Amount, 64)
		// Prefer the on-chain ledger close time so history is ordered by when the
		// payment actually happened, not when our sync happened to run.
		createdAt := p.At
		if createdAt == "" {
			createdAt = time.Now().UTC().Format(time.RFC3339)
		}
		tx := Transaction{
			ID:           uuid.NewString(),
			WalletID:     w.ID,
			UserID:       w.UserID,
			Type:         "in",
			Counterparty: p.From,
			Amount:       strconv.FormatFloat(amt, 'f', 4, 64),
			Symbol:       p.Symbol,
			Status:       "confirmed",
			TxHash:       p.Hash,
			CreatedAt:    createdAt,
		}
		s.db.Exec(
			`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, memo, status, signature, envelope_xdr, tx_hash, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, '', ?, '', '', ?, ?)`,
			tx.ID, tx.WalletID, tx.UserID, tx.Type, tx.Counterparty, tx.Amount, tx.Symbol, tx.Status, tx.TxHash, tx.CreatedAt)
		s.hub.publishTxn(w.UserID, tx)
		added++
	}
	if added > 0 {
		s.refreshWallet(w.ID)
	}
	return added
}

// syncWallet is the HTTP handler that triggers an incoming-payment sync.
func (s *server) syncWallet(c *gin.Context) {
	w, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || w.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"synced": s.syncIncoming(w)})
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

const walletCols = `id, name, chain, symbol, address, pubkey, balance, status, created_at`

func scanWallet(sc interface{ Scan(...any) error }) (Wallet, error) {
	var w Wallet
	err := sc.Scan(&w.ID, &w.Name, &w.Chain, &w.Symbol, &w.Address, &w.Pubkey, &w.Balance, &w.Status, &w.CreatedAt)
	return w, err
}

func (s *server) listWallets(c *gin.Context) {
	userID := c.GetString("userID")
	rows, err := s.db.Query(
		`SELECT `+walletCols+` FROM wallets WHERE user_id = ? ORDER BY created_at`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()

	wallets := []Wallet{}
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		wallets = append(wallets, w)
	}
	c.JSON(http.StatusOK, wallets)
}

type createWalletBody struct {
	Name  string `json:"name"`
	Chain string `json:"chain"`
}

// createWallet inserts a pending wallet then triggers a distributed keygen.
// The address/pubkey arrive asynchronously via the MPC result (pushed over SSE).
func (s *server) createWallet(c *gin.Context) {
	userID := c.GetString("userID")
	var body createWalletBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	meta, ok := chainMeta[body.Chain]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported chain"})
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = capitalize(body.Chain) + " Wallet"
	}

	w := Wallet{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		Chain:     body.Chain,
		Symbol:    meta.Symbol,
		Address:   "",
		Balance:   "0.00",
		Status:    "generating",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_, err := s.db.Exec(
		`INSERT INTO wallets (id, user_id, name, chain, symbol, address, pubkey, balance, status, created_at)
		 VALUES (?, ?, ?, ?, ?, '', '', ?, ?, ?)`,
		w.ID, w.UserID, w.Name, w.Chain, w.Symbol, w.Balance, w.Status, w.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}

	if err := s.mpc.CreateWallet(w.ID); err != nil {
		s.db.Exec(`UPDATE wallets SET status = 'failed' WHERE id = ?`, w.ID)
		c.JSON(http.StatusBadGateway, gin.H{"error": "keygen dispatch failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, w)
}

func (s *server) getWallet(c *gin.Context) {
	w, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

func (s *server) walletFor(userID, id string) (Wallet, bool) {
	w, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletCols+` FROM wallets WHERE id = ? AND user_id = ?`, id, userID))
	if err != nil {
		return Wallet{}, false
	}
	w.UserID = userID
	return w, true
}

// walletBalance fetches the live on-chain balance and an explorer link.
func (s *server) walletBalance(c *gin.Context) {
	w, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || w.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"assets":   getBalances(w.Chain, w.Address),
		"symbol":   w.Symbol,
		"explorer": explorerAddress(w.Chain, w.Address),
	})
	// The detail view just read the live balance; sync the cached column and
	// push an SSE update so the wallet list reflects it immediately instead of
	// waiting for the next 30s poll.
	go s.refreshWallet(w.ID)
}

// fundWallet requests testnet funds (Friendbot / airdrop).
func (s *server) fundWallet(c *gin.Context) {
	w, ok := s.walletFor(c.GetString("userID"), c.Param("id"))
	if !ok || w.Status != "ready" {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not ready"})
		return
	}
	if err := fund(w.Chain, w.Address); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	bal, _ := getBalance(w.Chain, w.Address)
	// Persist immediately so the wallet list reflects it without waiting for the refresher.
	s.db.Exec(`UPDATE wallets SET balance = ? WHERE id = ?`, bal, w.ID)
	if ww, ok := s.walletByID(w.ID); ok {
		s.hub.publishWallet(c.GetString("userID"), ww)
	}

	// Pull the faucet payment from chain history as a real incoming tx.
	s.syncIncoming(w)

	c.JSON(http.StatusOK, gin.H{"balance": bal})
}

// deleteWallet removes a wallet and its transactions from our records.
// (The MPC key shares themselves stay on the cluster nodes.)
func (s *server) deleteWallet(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")
	if _, ok := s.walletFor(userID, id); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}
	s.db.Exec(`DELETE FROM transactions WHERE wallet_id = ? AND user_id = ?`, id, userID)
	s.db.Exec(`DELETE FROM wallets WHERE id = ? AND user_id = ?`, id, userID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) walletByID(id string) (Wallet, bool) {
	w, err := scanWallet(s.db.QueryRow(
		`SELECT `+walletCols+` FROM wallets WHERE id = ?`, id))
	if err != nil {
		return Wallet{}, false
	}
	return w, true
}
