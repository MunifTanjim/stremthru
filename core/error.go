package core

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
)

type ErrorType = server.LegacyErrorType

const (
	ErrorTypeAPI      = server.LegacyErrorTypeAPI
	ErrorTypeStore    = server.LegacyErrorTypeStore
	ErrorTypeUpstream = server.LegacyErrorTypeUpstream
	ErrorTypeUnknown  = server.LegacyErrorTypeUnknown
)

type ErrorCode = server.ErrorCode

const (
	ErrorCodeUnknown                     = server.ErrorCodeUnknown
	ErrorCodeBadGateway                  = server.ErrorCodeBadGateway
	ErrorCodeBadRequest                  = server.ErrorCodeBadRequest
	ErrorCodeConflict                    = server.ErrorCodeConflict
	ErrorCodeForbidden                   = server.ErrorCodeForbidden
	ErrorCodeGone                        = server.ErrorCodeGone
	ErrorCodeInternalServerError         = server.ErrorCodeInternalServerError
	ErrorCodeMethodNotAllowed            = server.ErrorCodeMethodNotAllowed
	ErrorCodeNotFound                    = server.ErrorCodeNotFound
	ErrorCodeNotImplemented              = server.ErrorCodeNotImplemented
	ErrorCodePaymentRequired             = server.ErrorCodePaymentRequired
	ErrorCodeProxyAuthenticationRequired = server.ErrorCodeProxyAuthenticationRequired
	ErrorCodeServiceUnavailable          = server.ErrorCodeServiceUnavailable
	ErrorCodeTooManyRequests             = server.ErrorCodeTooManyRequests
	ErrorCodeUnauthorized                = server.ErrorCodeUnauthorized
	ErrorCodeUnavailableForLegalReasons  = server.ErrorCodeUnavailableForLegalReasons
	ErrorCodeUnprocessableEntity         = server.ErrorCodeUnprocessableEntity
	ErrorCodeUnsupportedMediaType        = server.ErrorCodeUnsupportedMediaType
	ErrorCodeStoreLimitExceeded          = server.ErrorCodeStoreLimitExceeded
	ErrorCodeStoreMagnetInvalid          = server.ErrorCodeStoreMagnetInvalid
	ErrorCodeStoreNameInvalid            = server.ErrorCodeStoreNameInvalid
	ErrorCodeStoreServerDown             = server.ErrorCodeStoreServerDown
)

type StremThruError = server.StremThruError

type Error = server.LegacyError

func NewError(msg string) *Error {
	err := &Error{}
	err.Type = ErrorTypeUnknown
	err.Msg = msg
	return err
}

type err = Error

type APIError struct {
	err
}

func NewAPIError(msg string) *APIError {
	err := &APIError{}
	err.Type = ErrorTypeAPI
	err.Msg = msg
	return err
}

type StoreError struct {
	err
}

func NewStoreError(msg string) *StoreError {
	err := &StoreError{}
	err.Type = ErrorTypeStore
	err.Msg = msg
	return err
}

type UpstreamError struct {
	err
}

func NewUpstreamError(msg string) *UpstreamError {
	err := &UpstreamError{}
	err.Type = ErrorTypeUpstream
	err.Msg = msg
	return err
}

var PackError = server.PackLegacyError

func LogError(r *http.Request, msg string, err error) {
	ctx := server.GetReqCtx(r)
	ctx.Log.Error(msg, "error", PackError(err))
}
