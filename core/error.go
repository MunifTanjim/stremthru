package core

import (
	"encoding/json"
	"net/http"
)

type ErrorType string

const (
	ErrorTypeAPI      ErrorType = "api_error"
	ErrorTypeStore    ErrorType = "store_error"
	ErrorTypeUpstream ErrorType = "upstream_error"
	ErrorTypeUnknown  ErrorType = "unknown_error"
)

type ErrorCode string

const (
	ErrorCodeUnknown ErrorCode = "UNKNOWN"

	ErrorCodeBadGateway                  ErrorCode = "BAD_GATEWAY"
	ErrorCodeBadRequest                  ErrorCode = "BAD_REQUEST"
	ErrorCodeConflict                    ErrorCode = "CONFLICT"
	ErrorCodeForbidden                   ErrorCode = "FORBIDDEN"
	ErrorCodeGone                        ErrorCode = "GONE"
	ErrorCodeInternalServerError         ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrorCodeMethodNotAllowed            ErrorCode = "METHOD_NOT_ALLOWED"
	ErrorCodeNotFound                    ErrorCode = "NOT_FOUND"
	ErrorCodeNotImplemented              ErrorCode = "NOT_IMPLEMENTED"
	ErrorCodePaymentRequired             ErrorCode = "PAYMENT_REQUIRED"
	ErrorCodeProxyAuthenticationRequired ErrorCode = "PROXY_AUTHENTICATION_REQUIRED"
	ErrorCodeServiceUnavailable          ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeTooManyRequests             ErrorCode = "TOO_MANY_REQUESTS"
	ErrorCodeUnauthorized                ErrorCode = "UNAUTHORIZED"
	ErrorCodeUnavailableForLegalReasons  ErrorCode = "UNAVAILABLE_FOR_LEGAL_REASONS"
	ErrorCodeUnprocessableEntity         ErrorCode = "UNPROCESSABLE_ENTITY"
	ErrorCodeUnsupportedMediaType        ErrorCode = "UNSUPPORTED_MEDIA_TYPE"

	ErrorCodeStoreLimitExceeded ErrorCode = "STORE_LIMIT_EXCEEDED"
	ErrorCodeStoreMagnetInvalid ErrorCode = "STORE_MAGNET_INVALID"
	ErrorCodeStoreNameInvalid   ErrorCode = "STORE_NAME_INVALID"
)

type StremThruError interface {
	Pack()
	GetStatusCode() int
	GetError() *Error
}

type Error struct {
	Type ErrorType `json:"type"`

	Code ErrorCode `json:"code,omitempty"`
	Msg  string    `json:"message"`

	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`

	StoreName     string `json:"store_name,omitempty"`
	UpstreamCause error  `json:"__upstream_cause__,omitempty"`

	Cause error `json:"__cause__,omitempty"`
}

func (e *Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) InjectReq(r *http.Request) {
	e.Method = r.Method
	e.Path = r.URL.Path
	if storeName := r.Header.Get("X-StremThru-Store-Name"); storeName != "" {
		e.StoreName = storeName
	}
}

var errorCodeByStatusCode = map[int]ErrorCode{
	http.StatusBadGateway:                 ErrorCodeBadGateway,
	http.StatusBadRequest:                 ErrorCodeBadRequest,
	http.StatusConflict:                   ErrorCodeConflict,
	http.StatusForbidden:                  ErrorCodeForbidden,
	http.StatusGone:                       ErrorCodeGone,
	http.StatusInternalServerError:        ErrorCodeInternalServerError,
	http.StatusMethodNotAllowed:           ErrorCodeMethodNotAllowed,
	http.StatusNotFound:                   ErrorCodeNotFound,
	http.StatusNotImplemented:             ErrorCodeNotImplemented,
	http.StatusPaymentRequired:            ErrorCodePaymentRequired,
	http.StatusProxyAuthRequired:          ErrorCodeProxyAuthenticationRequired,
	http.StatusServiceUnavailable:         ErrorCodeServiceUnavailable,
	http.StatusTooManyRequests:            ErrorCodeTooManyRequests,
	http.StatusUnauthorized:               ErrorCodeUnauthorized,
	http.StatusUnavailableForLegalReasons: ErrorCodeUnavailableForLegalReasons,
	http.StatusUnprocessableEntity:        ErrorCodeUnprocessableEntity,
	http.StatusUnsupportedMediaType:       ErrorCodeUnsupportedMediaType,
}

func (e *Error) Pack() {
	if e.StatusCode == 0 {
		e.StatusCode = 500
	}
	if e.Code == "" {
		if errorCode, found := errorCodeByStatusCode[e.StatusCode]; found {
			e.Code = errorCode
		}
	}
	if e.Msg == "" {
		if e.Cause != nil {
			e.Msg = e.Cause.Error()
		} else if e.UpstreamCause != nil {
			e.Msg = e.UpstreamCause.Error()
		} else {
			e.Msg = http.StatusText(e.StatusCode)
		}
	}
}

func (e *Error) GetStatusCode() int {
	return e.StatusCode
}

func (e *Error) GetError() *Error {
	return e
}

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
