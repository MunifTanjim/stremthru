package core

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
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
	Pack(r *http.Request)
	GetStatusCode() int
	GetError() *Error
	Send(w http.ResponseWriter, r *http.Request)
}

type Error struct {
	RequestId string `json:"request_id"`

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

type errorResponse struct {
	Error *Error `json:"error,omitempty"`
}

func (e *Error) LogValue() slog.Value {
	attrs := []slog.Attr{}
	if e.Type != "" {
		attrs = append(attrs, slog.String("type", string(e.Type)))
	}
	if e.Code != "" {
		attrs = append(attrs, slog.String("code", string(e.Code)))
	}
	if e.Msg != "" {
		attrs = append(attrs, slog.String("message", e.Msg))
	}
	if e.Method != "" {
		attrs = append(attrs, slog.String("method", e.Method))
	}
	if e.Path != "" {
		attrs = append(attrs, slog.String("path", e.Path))
	}
	if e.StatusCode != 0 {
		attrs = append(attrs, slog.Int("status_code", e.StatusCode))
	}
	if e.StoreName != "" {
		attrs = append(attrs, slog.String("store_name", e.StoreName))
	}
	if e.UpstreamCause != nil {
		attrs = append(attrs, slog.Any("upstream_cause", e.UpstreamCause))
	}
	if e.Cause != nil {
		attrs = append(attrs, slog.Any("cause", e.Cause))
	}
	return slog.GroupValue(attrs...)
}

func (e *Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) GetStatusCode() int {
	return e.StatusCode
}

func (e *Error) GetError() *Error {
	return e
}

func (e *Error) WithCause(cause error) *Error {
	e.Cause = cause
	return e
}

func (e *Error) Send(w http.ResponseWriter, r *http.Request) {
	e.Pack(r)

	ctx := server.GetReqCtx(r)
	ctx.Error = e

	res := &errorResponse{Error: e}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.GetStatusCode())
	if err := json.NewEncoder(w).Encode(res); err != nil {
		LogError(r, "failed to encode json", err)
	}
}

func (e *Error) InjectReq(r *http.Request) {
	e.RequestId = r.Header.Get("Request-ID")
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

var statusCodeByErrorCode = map[ErrorCode]int{
	ErrorCodeBadGateway:                  http.StatusBadGateway,
	ErrorCodeBadRequest:                  http.StatusBadRequest,
	ErrorCodeConflict:                    http.StatusConflict,
	ErrorCodeForbidden:                   http.StatusForbidden,
	ErrorCodeGone:                        http.StatusGone,
	ErrorCodeInternalServerError:         http.StatusInternalServerError,
	ErrorCodeMethodNotAllowed:            http.StatusMethodNotAllowed,
	ErrorCodeNotFound:                    http.StatusNotFound,
	ErrorCodeNotImplemented:              http.StatusNotImplemented,
	ErrorCodePaymentRequired:             http.StatusPaymentRequired,
	ErrorCodeProxyAuthenticationRequired: http.StatusProxyAuthRequired,
	ErrorCodeServiceUnavailable:          http.StatusServiceUnavailable,
	ErrorCodeTooManyRequests:             http.StatusTooManyRequests,
	ErrorCodeUnauthorized:                http.StatusUnauthorized,
	ErrorCodeUnavailableForLegalReasons:  http.StatusUnavailableForLegalReasons,
	ErrorCodeUnprocessableEntity:         http.StatusUnprocessableEntity,
	ErrorCodeUnsupportedMediaType:        http.StatusUnsupportedMediaType,

	ErrorCodeStoreMagnetInvalid: http.StatusBadRequest,
	ErrorCodeStoreNameInvalid:   http.StatusBadRequest,
}

func (e *Error) Pack(r *http.Request) {
	if e.StatusCode == 0 {
		e.StatusCode = 500
	}
	if e.Code == "" {
		if errorCode, found := errorCodeByStatusCode[e.StatusCode]; found {
			e.Code = errorCode
		}
	}
	if statusCode, found := statusCodeByErrorCode[e.Code]; found && statusCode != e.StatusCode {
		e.StatusCode = statusCode
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
	if r != nil {
		if e.RequestId == "" {
			e.RequestId = r.Header.Get(server.HEADER_REQUEST_ID)
		}
	}
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

func PackError(err error) error {
	var e StremThruError
	if sterr, ok := err.(StremThruError); ok {
		e = sterr
	} else {
		e = &Error{Cause: err}
	}
	e.Pack(nil)
	return e.GetError()
}

func LogError(r *http.Request, msg string, err error) {
	ctx := server.GetReqCtx(r)
	ctx.Log.Error(msg, "error", PackError(err))
}
