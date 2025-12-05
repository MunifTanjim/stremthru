package stremio_watched_bitfield

import "fmt"

type ErrorCode string

const (
	ErrCodeInvalidFormat ErrorCode = "invalid_format"
	ErrCodeUnexpected    ErrorCode = "unexpected"
)

type Error struct {
	Code    ErrorCode
	Message string

	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) WithCause(cause error) error {
	e.cause = cause
	return e
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}
