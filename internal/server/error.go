package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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
	ErrorCodeLocked                      ErrorCode = "LOCKED"
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
	ErrorCodeStoreServerDown    ErrorCode = "STORE_SERVER_DOWN"
)

type ErrorDomain = string

var (
	ErrorDomainStore ErrorDomain = "store"
)

type ErrorLocationType = string

var (
	LocationTypeBody   ErrorLocationType = "body"
	LocationTypeCookie ErrorLocationType = "cookie"
	LocationTypeHeader ErrorLocationType = "header"
	LocationTypePath   ErrorLocationType = "path"
	LocationTypeQuery  ErrorLocationType = "query"
)

type Error struct {
	Domain       string `json:"domain,omitempty"`
	ExtendedHelp string `json:"extendedHelp,omitempty"`
	Location     string `json:"location,omitempty"`
	LocationType string `json:"locationType,omitempty"`
	Message      string `json:"message"`
	Reason       string `json:"reason,omitempty"`
	SendReport   string `json:"sendReport,omitempty"`
}

type APIError struct {
	StatusCode int `json:"-"`

	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Errors  []Error   `json:"errors"`

	Cause     error  `json:"-"`
	Method    string `json:"-"`
	Path      string `json:"-"`
	RequestId string `json:"-"`
	Type      string `json:"-"`

	meta map[string]any `json:"-"`
}

func (e *APIError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e *APIError) WithCode(code ErrorCode) *APIError {
	e.Code = code
	return e
}

func (e *APIError) WithMessage(message string) *APIError {
	e.Message = message
	return e
}
func (e *APIError) WithCause(cause ...error) *APIError {
	e.Cause = errors.Join(cause...)
	return e
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

func (e *APIError) Set(key string, value any) *APIError {
	if e.meta == nil {
		e.meta = make(map[string]any)
	}
	e.meta[key] = value
	return e
}

func (e *APIError) LogValue() slog.Value {
	attrs := []slog.Attr{}
	if e.StatusCode != 0 {
		attrs = append(attrs, slog.Int("status_code", e.StatusCode))
	}
	if e.Code != "" {
		attrs = append(attrs, slog.String("code", string(e.Code)))
	}
	if e.Message != "" {
		attrs = append(attrs, slog.String("message", e.Message))
	}

	if e.Cause != nil {
		attrs = append(attrs, slog.Any("cause", e.Cause))
	}
	if e.Method != "" {
		attrs = append(attrs, slog.String("method", e.Method))
	}
	if e.Path != "" {
		attrs = append(attrs, slog.String("path", e.Path))
	}
	if e.Type != "" {
		attrs = append(attrs, slog.String("type", string(e.Type)))
	}
	for k, v := range e.meta {
		attrs = append(attrs, slog.Any(k, v))
	}
	return slog.GroupValue(attrs...)
}

func (e *APIError) InjectRequest(r *http.Request) {
	if r == nil {
		return
	}
	e.RequestId = r.Header.Get(HEADER_REQUEST_ID)
	e.Method = r.Method
	e.Path = r.URL.Path
}

func (e *APIError) Send(w http.ResponseWriter, r *http.Request) {
	if len(e.Errors) == 0 {
		e.Append(Error{
			Message: e.Message,
		})
	}
	SendError(w, r, e)
}

func (e *APIError) Append(errs ...Error) *APIError {
	e.Errors = append(e.Errors, errs...)
	return e
}

func NewAPIError(statusCode int, message string, code ErrorCode, errors ...Error) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Errors:     errors,
	}
}

func ErrorBadRequest(r *http.Request) *APIError {
	err := NewAPIError(http.StatusBadRequest, "Bad Request", ErrorCodeBadRequest)
	err.InjectRequest(r)
	return err
}

func ErrorUnauthorized(r *http.Request) *APIError {
	err := NewAPIError(http.StatusUnauthorized, "Unauthorized", ErrorCodeUnauthorized)
	err.InjectRequest(r)
	return err
}

func ErrorForbidden(r *http.Request) *APIError {
	err := NewAPIError(http.StatusForbidden, "Forbidden", ErrorCodeForbidden)
	err.InjectRequest(r)
	return err
}

func ErrorNotFound(r *http.Request) *APIError {
	err := NewAPIError(http.StatusNotFound, "Not Found", ErrorCodeNotFound)
	err.InjectRequest(r)
	return err
}

func ErrorMethodNotAllowed(r *http.Request) *APIError {
	err := NewAPIError(http.StatusMethodNotAllowed, "Method Not Allowed", ErrorCodeMethodNotAllowed)
	err.InjectRequest(r)
	return err
}

func ErrorUnsupportedMediaType(r *http.Request) *APIError {
	err := NewAPIError(http.StatusUnsupportedMediaType, "Unsupported Media Type", ErrorCodeUnsupportedMediaType)
	err.InjectRequest(r)
	return err
}

func ErrorLocked(r *http.Request) *APIError {
	err := NewAPIError(http.StatusLocked, "Locked", ErrorCodeLocked)
	err.InjectRequest(r)
	return err
}

func ErrorInternalServerError(r *http.Request) *APIError {
	err := NewAPIError(http.StatusInternalServerError, "Internal Server Error", ErrorCodeInternalServerError)
	err.InjectRequest(r)
	return err
}
