package service

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/price"

	"go.uber.org/zap"
)

// PriceSource values payments in USD for the approval policy.
type PriceSource interface {
	Quote(code, issuer string) (price.Quote, error)
}

var usdAmount = regexp.MustCompile(`^\d{1,12}(\.\d{1,2})?$`)

// parseUSD validates a positive dollar amount with at most 2 decimals and returns it without trailing zeros.
func parseUSD(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	value, ok := new(big.Rat).SetString(raw)
	if !usdAmount.MatchString(raw) || !ok || value.Sign() <= 0 {
		return "", apperror.InvalidApprovalThreshold.New()
	}
	return trimAmount(value.FloatString(2)), nil
}

// valuation is what a payment was worth when the approval decision was made.
type valuation struct {
	usd      *big.Rat
	usdText  string
	price    string
	pricedAt string
}

// value prices a payment; ok is false when the asset has no fresh price.
func value(prices PriceSource, payment preparedPayment) (valuation, bool) {
	quote, err := prices.Quote(payment.AssetCode, payment.AssetIssuer)
	if err != nil {
		zap.L().Warn("no USD price for payment, holding it for approval", zap.String("asset", payment.Symbol), zap.Error(err))
		return valuation{}, false
	}
	units, ok := new(big.Rat).SetString(payment.Amount)
	perUnit, priceOK := new(big.Rat).SetString(quote.Price)
	if !ok || !priceOK {
		zap.L().Warn("unreadable payment amount or price", zap.String("amount", payment.Amount), zap.String("price", quote.Price))
		return valuation{}, false
	}
	usd := new(big.Rat).Mul(units, perUnit)
	return valuation{
		usd:      usd,
		usdText:  usd.FloatString(2),
		price:    trimAmount(quote.Price),
		pricedAt: quote.FetchedAt.UTC().Format(time.RFC3339),
	}, true
}

// convertXLMToUSD turns a legacy XLM threshold into dollars, never rounding down to zero.
func convertXLMToUSD(thresholdXLM, xlmPrice string) (string, error) {
	units, ok := new(big.Rat).SetString(thresholdXLM)
	perUnit, priceOK := new(big.Rat).SetString(xlmPrice)
	if !ok || !priceOK {
		return "", fmt.Errorf("convert %q XLM at %q", thresholdXLM, xlmPrice)
	}
	usd := new(big.Rat).Mul(units, perUnit).FloatString(2)
	if usd == "0.00" {
		usd = "0.01"
	}
	return trimAmount(usd), nil
}
