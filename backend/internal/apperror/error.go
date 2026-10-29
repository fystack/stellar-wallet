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

// Spec values live only in catalog.go, never at call sites.
type Spec struct {
	kind    Kind
	code    string
	message string
}

// Error keeps the safe public message apart from the technical cause.
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
