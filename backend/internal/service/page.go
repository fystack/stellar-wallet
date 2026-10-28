package service

import (
	"encoding/base64"
	"strings"

	"stellar-wallet-backend/internal/apperror"
	"stellar-wallet-backend/internal/repository"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type PageInput struct {
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
}

// Page.Next is the Cursor for the following page; empty on the last page.
type Page[T any] struct {
	Items []T
	Next  string
}

func (input PageInput) resolve() (repository.Cursor, int, error) {
	limit := input.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 0 || limit > maxPageSize {
		return repository.Cursor{}, 0, apperror.InvalidPage.New()
	}
	if input.Cursor == "" {
		return repository.Cursor{}, limit, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(input.Cursor)
	if err != nil {
		return repository.Cursor{}, 0, apperror.InvalidPage.New()
	}
	createdAt, id, ok := strings.Cut(string(decoded), "|")
	if !ok || createdAt == "" || id == "" {
		return repository.Cursor{}, 0, apperror.InvalidPage.New()
	}
	return repository.Cursor{CreatedAt: createdAt, ID: id}, limit, nil
}

// pageOf expects up to limit+1 rows; the extra row only signals that another page exists.
func pageOf[T any](rows []T, limit int, cursor func(T) repository.Cursor) Page[T] {
	if len(rows) <= limit {
		return Page[T]{Items: rows}
	}
	rows = rows[:limit]
	last := cursor(rows[limit-1])
	return Page[T]{Items: rows, Next: base64.RawURLEncoding.EncodeToString([]byte(last.CreatedAt + "|" + last.ID))}
}
