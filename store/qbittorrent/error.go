package qbittorrent

import (
	"encoding/json"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type QbitError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func (e *QbitError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func newQbitError(statusCode int, body []byte) *QbitError {
	msg := string(body)
	if msg == "" {
		msg = http.StatusText(statusCode)
	}
	return &QbitError{
		StatusCode: statusCode,
		Message:    msg,
	}
}

func TranslateStatusCode(statusCode int) core.ErrorCode {
	switch {
	case statusCode == http.StatusForbidden:
		return core.ErrorCodeUnauthorized
	case statusCode == http.StatusNotFound:
		return core.ErrorCodeNotFound
	case statusCode == http.StatusConflict:
		return core.ErrorCodeConflict
	case statusCode >= 500:
		return core.ErrorCodeServiceUnavailable
	case statusCode >= 400:
		return core.ErrorCodeBadRequest
	default:
		return core.ErrorCodeUnknown
	}
}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameQBittorrent)

	if qerr, ok := cause.(*QbitError); ok {
		err.Msg = qerr.Message
		err.Code = TranslateStatusCode(qerr.StatusCode)
		err.StatusCode = qerr.StatusCode
		err.UpstreamCause = qerr
	} else {
		err.Cause = cause
	}

	return err
}
