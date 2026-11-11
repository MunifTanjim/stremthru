package highway

import (
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameHighWay)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Message
		err.StatusCode = rerr.StatusCode
		err.Code = errorCode(rerr)
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}

// errorCode maps HighWay errors ({"error": "<code>", "message": "..."}) to StremThru codes.
func errorCode(err *ResponseError) core.ErrorCode {
	switch err.StatusCode {
	case http.StatusUnauthorized:
		return core.ErrorCodeUnauthorized
	case http.StatusForbidden:
		return core.ErrorCodeForbidden
	case http.StatusNotFound:
		return core.ErrorCodeNotFound
	case http.StatusTooManyRequests:
		return core.ErrorCodeTooManyRequests
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return core.ErrorCodeStoreServerDown
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		if strings.Contains(err.ErrorCode, "magnet") || strings.Contains(err.ErrorCode, "torrent") || err.ErrorCode == "invalid_check" {
			return core.ErrorCodeStoreMagnetInvalid
		}
		return core.ErrorCodeBadRequest
	default:
		if err.StatusCode >= http.StatusInternalServerError {
			return core.ErrorCodeInternalServerError
		}
		return core.ErrorCodeUnknown
	}
}
