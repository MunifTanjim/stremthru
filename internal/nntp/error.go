package nntp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
	"syscall"
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
	Msg        string
	Message    string
	StatusCode int
	Cause      error
}

func (e *Error) WithCause(err error) *Error {
	if tperr, ok := err.(*textproto.Error); ok && tperr.Code == e.StatusCode && tperr.Msg == e.Msg {
		return e
	}
	e.Cause = err
	return e
}

func (e *Error) Error() string {
	var err strings.Builder
	err.WriteString(string(e.Code))
	if e.StatusCode > 0 {
		err.WriteString(fmt.Sprintf(" (%d %s)", e.StatusCode, e.Msg))
	}
	if e.Message != "" {
		err.WriteString(" ")
		err.WriteString(e.Message)
	}
	if e.Cause != nil {
		err.WriteString(": ")
		err.WriteString(e.Cause.Error())
	}
	return err.String()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) isAuthError() bool {
	if e.Code == ErrorCodeAuthentication || e.Code == ErrorCodeAuthRequired {
		return true
	}
	message := strings.ToLower(e.Message)
	if strings.Contains(message, "authentication failed") {
		return true
	}
	return false
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
	case StatusServiceTemporarilyUnavailable, StatusServicePermanentlyUnavailable:
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
	case StatusUnknownCommand, StatusSyntaxError, StatusFeatureNotSupported, StatusBase64EncodingError:
		code = ErrorCodeServerError
	}

	return &Error{
		Code:       code,
		Msg:        message,
		Message:    fmt.Sprintf("%s failed", cmd),
		StatusCode: statusCode,
	}
}

func isConnectionError(err error) bool {
	if errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var protocolErr textproto.ProtocolError
	if errors.As(err, &protocolErr) {
		return true
	}

	var nntpErr *Error
	if ok := errors.As(err, &nntpErr); ok {
		switch nntpErr.StatusCode {
		case StatusServiceTemporarilyUnavailable, StatusServicePermanentlyUnavailable:
			return true
		default:
			return false
		}
	}

	return false
}

var (
	ErrPoolNotOnline = errors.New("Pool is not online")
)
