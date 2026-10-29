package service

import (
	"strings"

	"stellar-wallet-backend/internal/apperror"

	"github.com/stellar/go/amount"
)

// parseAmount validates a positive Stellar amount (at most 7 decimals) and returns it exactly, without trailing zeros.
func parseAmount(raw string) (string, error) {
	stroops, err := amount.ParseInt64(strings.TrimSpace(raw))
	if err != nil || stroops <= 0 {
		return "", apperror.InvalidAmount.New()
	}
	return trimAmount(amount.StringFromInt64(stroops)), nil
}

func trimAmount(value string) string {
	if !strings.Contains(value, ".") {
		return value
	}
	return strings.TrimSuffix(strings.TrimRight(value, "0"), ".")
}
