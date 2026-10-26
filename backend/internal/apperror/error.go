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
)

// Error separates a safe public message from the underlying technical cause.
// Callers can add context with fmt.Errorf("operation: %w", err).
type Error struct {
	Kind    Kind
	Code    string
	Message string
	cause   error
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(cause error, kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message, cause: cause}
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }
