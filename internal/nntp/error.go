package nntp

import (
	"fmt"
	"strings"
)

type ErrorCode string

const (
	// Connection errors (client-side network issues)
	ErrorCodeConnection ErrorCode = "NNTP_CONNECTION_ERROR"

	// Protocol errors (invalid server responses, RFC 3977 Section 3)
	ErrorCodeProtocol ErrorCode = "NNTP_PROTOCOL_ERROR"

	// Service unavailable (RFC 3977 status codes 400, 502)
	ErrorCodeServiceUnavailable ErrorCode = "NNTP_SERVICE_UNAVAILABLE"

	// Authentication errors (RFC 4643 status codes 481, 482)
	ErrorCodeAuthentication ErrorCode = "NNTP_AUTH_ERROR"

	// Authentication required (RFC 4643 status code 480)
	ErrorCodeAuthRequired ErrorCode = "NNTP_AUTH_REQUIRED"

	// Encryption required (RFC 4643 status code 483)
	ErrorCodeEncryptionRequired ErrorCode = "NNTP_ENCRYPTION_REQUIRED"

	// No such group error (RFC 3977 status code 411)
	ErrorCodeNoSuchGroup ErrorCode = "NNTP_NO_SUCH_GROUP"

	// No group selected (RFC 3977 status code 412)
	ErrorCodeNoGroupSelected ErrorCode = "NNTP_NO_GROUP_SELECTED"

	// No such article error (RFC 3977 status codes 420, 421, 422, 423, 430)
	ErrorCodeNoSuchArticle ErrorCode = "NNTP_NO_SUCH_ARTICLE"

	// Posting failed (RFC 3977 status codes 440, 441)
	ErrorCodePostingFailed ErrorCode = "NNTP_POSTING_FAILED"

	// Transfer failed (RFC 3977 status codes 435, 436, 437)
	ErrorCodeTransferFailed ErrorCode = "NNTP_TRANSFER_FAILED"

	// Command failed (RFC 3977 Section 3.2.1)
	ErrorCodeCommandFailed ErrorCode = "NNTP_COMMAND_FAILED"

	// Server errors (RFC 3977 status codes 500, 501, 503, 504)
	ErrorCodeServerError ErrorCode = "NNTP_SERVER_ERROR"
)

type Error struct {
	Code       ErrorCode
	Message    string
	StatusCode int
	Cause      error
}

func (e *Error) WithCause(err error) *Error {
	e.Cause = err
	return e
}

func (e *Error) Error() string {
	var err strings.Builder
	err.WriteString(string(e.Code))
	if e.Cause != nil {
		err.WriteString(": ")
		err.WriteString(e.Cause.Error())
	}
	if e.StatusCode > 0 {
		err.WriteString(fmt.Sprintf(" (status code: %d)", e.StatusCode))
	}
	return err.String()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func NewConnectionError(msg string) *Error {
	return &Error{
		Code:    ErrorCodeConnection,
		Message: msg,
	}
}

func NewProtocolError(statusCode int, message string) *Error {
	return &Error{
		Code:       ErrorCodeProtocol,
		Message:    message,
		StatusCode: statusCode,
	}
}

func NewCommandError(cmd string, statusCode int, message string) *Error {
	code := ErrorCodeCommandFailed

	switch statusCode {
	// Service unavailability
	case StatusServiceNotAvailable, StatusServiceUnavailable:
		code = ErrorCodeServiceUnavailable

	// Authentication and security
	case StatusAuthenticationRequired:
		code = ErrorCodeAuthRequired
	case StatusAuthenticationRejected, StatusAuthenticationOutOfSequence:
		code = ErrorCodeAuthentication
	case StatusEncryptionRequired:
		code = ErrorCodeEncryptionRequired

	// Group errors
	case StatusNoSuchGroup:
		code = ErrorCodeNoSuchGroup
	case StatusNoGroupSelected:
		code = ErrorCodeNoGroupSelected

	// Article errors
	case StatusNoCurrentArticle, StatusNoNextArticle, StatusNoPreviousArticle,
		StatusNoSuchArticle, StatusNoSuchArticleNumber:
		code = ErrorCodeNoSuchArticle

	// Posting errors
	case StatusPostingNotPermitted, StatusPostingFailed:
		code = ErrorCodePostingFailed

	// Transfer errors (IHAVE)
	case StatusArticleNotWanted, StatusTransferNotPossible, StatusTransferRejected:
		code = ErrorCodeTransferFailed

	// Server errors
	case StatusCommandNotRecognized, StatusSyntaxError, StatusFeatureNotSupported, StatusBase64EncodingError:
		code = ErrorCodeServerError
	}

	return &Error{
		Code:       code,
		Message:    fmt.Sprintf("%s failed: %s", cmd, message),
		StatusCode: statusCode,
	}
}
