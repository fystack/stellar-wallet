package service

import (
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/chain"
	"stellar-wallet-backend/internal/domain"
	"stellar-wallet-backend/internal/repository"

	"github.com/google/uuid"
)

const defaultSlippageBps = 100

type SwapChain interface {
	SwapQuote(send chain.SwapAsset, sendAmount string, dest chain.SwapAsset) (chain.SwapQuote, error)
	BuildSwap(wallet domain.Wallet, send chain.SwapAsset, sendAmount string, dest chain.SwapAsset, destMin string, path []chain.SwapAsset) (string, []byte, error)
}

type SwapService struct {
	wallets            repository.Store
	chain              SwapChain
	transactionService *TransactionService
}

func NewSwapService(wallets repository.Store, chain SwapChain, transactionService *TransactionService) *SwapService {
	return &SwapService{wallets: wallets, chain: chain, transactionService: transactionService}
}

type SwapInput struct {
	From        string `json:"from"` // "XLM" or "CODE:ISSUER"
	To          string `json:"to"`
	Amount      string `json:"amount"` // exact amount of From to send
	SlippageBps int    `json:"slippage_bps"`
}

type SwapResult struct {
	Transaction domain.Transaction `json:"transaction"`
	Quote       chain.SwapQuote    `json:"quote"`
	DestMin     string             `json:"dest_min"`
}

// Quote estimates a strict-send swap without signing anything.
func (s *SwapService) Quote(userID, walletID string, input SwapInput) (chain.SwapQuote, error) {
	wallet, err := ownedWallet(s.wallets, userID, walletID)
	if err != nil {
		return chain.SwapQuote{}, err
	}
	if wallet.Chain != "stellar" {
		return chain.SwapQuote{}, apperror.SwapUnsupportedChain.New()
	}
	amount, send, dest, err := parseSwap(input)
	if err != nil {
		return chain.SwapQuote{}, err
	}
	return s.quote(send, amount, dest)
}

// Create builds a slippage-protected PathPaymentStrictSend and dispatches it for signing.
func (s *SwapService) Create(userID, walletID string, input SwapInput) (SwapResult, error) {
	wallet, err := readyWallet(s.wallets, userID, walletID)
	if err != nil {
		return SwapResult{}, err
	}
	if wallet.Chain != "stellar" {
		return SwapResult{}, apperror.SwapUnsupportedChain.New()
	}
	amount, send, dest, err := parseSwap(input)
	if err != nil {
		return SwapResult{}, err
	}
	if err := s.transactionService.requireSigning(); err != nil {
		return SwapResult{}, err
	}
	quote, err := s.quote(send, amount, dest)
	if err != nil {
		return SwapResult{}, err
	}
	slippage := input.SlippageBps
	if slippage <= 0 {
		slippage = defaultSlippageBps
	}
	destMin := chain.ApplySlippage(quote.DestAmount, slippage)

	envelope, hash, err := s.chain.BuildSwap(wallet, send, amount, dest, destMin, quote.Path)
	if err != nil {
		return SwapResult{}, apperror.SwapBuildFailed.Wrap(err)
	}
	transaction := domain.Transaction{
		ID:           uuid.NewString(),
		WalletID:     wallet.ID,
		UserID:       userID,
		Type:         domain.TransactionSwap,
		Counterparty: wallet.Address, // strict-send lands back on the same account
		Amount:       amount,
		Symbol:       send.Code,
		RecvAmount:   quote.DestAmount,
		RecvSymbol:   dest.Code,
		Status:       domain.TransactionSigning,
		EnvelopeXDR:  envelope,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.transactionService.dispatch(wallet, transaction, hash); err != nil {
		return SwapResult{}, err
	}
	return SwapResult{Transaction: transaction, Quote: quote, DestMin: destMin}, nil
}

func (s *SwapService) quote(send chain.SwapAsset, amount string, dest chain.SwapAsset) (chain.SwapQuote, error) {
	quote, err := s.chain.SwapQuote(send, amount, dest)
	if err != nil {
		return chain.SwapQuote{}, apperror.SwapQuoteFailed.Wrap(err)
	}
	return quote, nil
}

func parseSwap(input SwapInput) (string, chain.SwapAsset, chain.SwapAsset, error) {
	amount, err := parseAmount(input.Amount)
	if err != nil {
		return "", chain.SwapAsset{}, chain.SwapAsset{}, err
	}
	send, dest := parseAsset(input.From), parseAsset(input.To)
	if send == dest {
		return "", chain.SwapAsset{}, chain.SwapAsset{}, apperror.InvalidAssetPair.New()
	}
	return amount, send, dest, nil
}

func parseAsset(raw string) chain.SwapAsset {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "XLM") {
		return chain.SwapAsset{Code: "XLM"}
	}
	code, issuer, _ := strings.Cut(raw, ":")
	return chain.SwapAsset{Code: code, Issuer: issuer}
}
