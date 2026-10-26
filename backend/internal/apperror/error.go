// Package apperror defines public business failures independently of HTTP.
package apperror

type Kind uint8

const (
	Internal Kind = iota
	Invalid
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Unavailable
	Upstream
	MethodNotAllowed
	TooManyRequests
)

// Spec is the single source of truth for a public API failure.
// Keep codes and user-facing messages in catalog.go, not at call sites.
type Spec struct {
	kind    Kind
	code    string
	message string
}

// Error separates a safe public message from the underlying technical cause.
// Callers can add context with fmt.Errorf("operation: %w", err).
type Error struct {
	Kind    Kind
	Code    string
	Message string
	cause   error
}

func (s Spec) New() *Error {
	return &Error{Kind: s.kind, Code: s.code, Message: s.message}
}

func (s Spec) Wrap(cause error) *Error {
	return &Error{Kind: s.kind, Code: s.code, Message: s.message, cause: cause}
}

func (s Spec) Code() string { return s.code }

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }
