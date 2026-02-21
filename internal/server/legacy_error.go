package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type StremThruError interface {
	Pack(r *http.Request)
	GetStatusCode() int
	GetError() *LegacyError
	Send(w http.ResponseWriter, r *http.Request)
}

type LegacyErrorType string

const (
	LegacyErrorTypeAPI      LegacyErrorType = "api_error"
	LegacyErrorTypeStore    LegacyErrorType = "store_error"
	LegacyErrorTypeUpstream LegacyErrorType = "upstream_error"
	LegacyErrorTypeUnknown  LegacyErrorType = "unknown_error"
)

type LegacyError struct {
	RequestId string `json:"request_id"`

	Type LegacyErrorType `json:"type"`

	Code ErrorCode `json:"code,omitempty"`
	Msg  string    `json:"message"`

	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`

	StoreName     string `json:"store_name,omitempty"`
	UpstreamCause error  `json:"__upstream_cause__,omitempty"`

	Cause error `json:"__cause__,omitempty"`
}

func (e *LegacyError) LogValue() slog.Value {
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

func (e LegacyError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (e LegacyError) Unwrap() error {
	return e.Cause
}

func (e *LegacyError) GetStatusCode() int {
	return e.StatusCode
}

func (e *LegacyError) GetError() *LegacyError {
	return e
}

func (e *LegacyError) WithCause(cause error) *LegacyError {
	e.Cause = cause
	return e
}

type errorResponse struct {
	Error *LegacyError `json:"error,omitempty"`
}

func (e *LegacyError) Send(w http.ResponseWriter, r *http.Request) {
	e.Pack(r)

	ctx := GetReqCtx(r)
	ctx.Error = e

	res := &errorResponse{Error: e}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.GetStatusCode())
	if err := json.NewEncoder(w).Encode(res); err != nil {
		ctx.Log.Error("failed to encode json", "error", PackLegacyError(err))
	}
}

func (e *LegacyError) InjectReq(r *http.Request) {
	if r == nil {
		return
	}
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

func (e *LegacyError) Pack(r *http.Request) {
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
			e.RequestId = r.Header.Get(HEADER_REQUEST_ID)
		}
	}
}

func PackLegacyError(err error) error {
	if err == nil {
		return nil
	}
	var e StremThruError
	if sterr, ok := err.(StremThruError); ok {
		e = sterr
	} else {
		e = &LegacyError{Cause: err}
	}
	e.Pack(nil)
	return e.GetError()
}
