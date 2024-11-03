package debridlink

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type ErrorCode = core.ErrorCode

const (
	ErrorCodeBadToken                ErrorCode = "badToken"
	ErrorCodeBadSign                 ErrorCode = "badSign"
	ErrorCodeHidedToken              ErrorCode = "hidedToken"
	ErrorCodeServerError             ErrorCode = "server_error"
	ErrorCodeAccessDenied            ErrorCode = "access_denied"
	ErrorCodeAuthorizationPending    ErrorCode = "authorization_pending"
	ErrorCodeUnsupportedGrantType    ErrorCode = "unsupported_grant_type"
	ErrorCodeUnsupportedResponseType ErrorCode = "unsupported_response_type"
	ErrorCodeInvalidRequest          ErrorCode = "invalid_request"
	ErrorCodeInvalidScope            ErrorCode = "invalid_scope"
	ErrorCodeExpiredToken            ErrorCode = "expired_token"
	ErrorCodeUnauthorizedClient      ErrorCode = "unauthorized_client"
	ErrorCodeInvalidClient           ErrorCode = "invalid_client"
	ErrorCodeUnknowR                 ErrorCode = "unknowR"
	ErrorCodeInternalError           ErrorCode = "internalError"
	ErrorCodeBadArguments            ErrorCode = "badArguments"
	ErrorCodeBadId                   ErrorCode = "badId"
	ErrorCodeFloodDetected           ErrorCode = "floodDetected"
	ErrorCodeServerNotAllowed        ErrorCode = "serverNotAllowed"
	ErrorCodeFreeServerOverload      ErrorCode = "freeServerOverload"
	ErrorCodeMaxAttempts             ErrorCode = "maxAttempts"
	ErrorCodeCaptchaRequired         ErrorCode = "captchaRequired"
	ErrorCodeAccountLocked           ErrorCode = "accountLocked"
	ErrorCodeNotDebrid               ErrorCode = "notDebrid"
	ErrorCodeHostNotValid            ErrorCode = "hostNotValid"
	ErrorCodeFileNotFound            ErrorCode = "fileNotFound"
	ErrorCodeFileNotAvailable        ErrorCode = "fileNotAvailable"
	ErrorCodeBadFileUrl              ErrorCode = "badFileUrl"
	ErrorCodeBadFilePassword         ErrorCode = "badFilePassword"
	ErrorCodeNotFreeHost             ErrorCode = "notFreeHost"
	ErrorCodeMaintenanceHost         ErrorCode = "maintenanceHost"
	ErrorCodeNoServerHost            ErrorCode = "noServerHost"
	ErrorCodeMaxLink                 ErrorCode = "maxLink"
	ErrorCodeMaxLinkHost             ErrorCode = "maxLinkHost"
	ErrorCodeMaxData                 ErrorCode = "maxData"
	ErrorCodeMaxDataHost             ErrorCode = "maxDataHost"
	ErrorCodeDisabledServerHost      ErrorCode = "disabledServerHost"
	ErrorCodeNotAddTorrent           ErrorCode = "notAddTorrent"
	ErrorCodeTorrentTooBig           ErrorCode = "torrentTooBig"
	ErrorCodeMaxTorrent              ErrorCode = "maxTorrent"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameDebridLink)

	if rerr, ok := cause.(*ResponseError); ok {
		if rerr.ErrDesc != "" {
			err.Msg = rerr.ErrDesc
		} else {
			err.Msg = "Debrid-Link Error Code: " + string(rerr.Err)
		}
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}

func UpstreamErrorFromRequest(cause error, req *http.Request, res *http.Response) error {
	err := UpstreamErrorWithCause(cause)
	err.InjectReq(req)
	if res != nil {
		err.StatusCode = res.StatusCode
	}
	if err.StatusCode <= http.StatusBadRequest {
		err.StatusCode = http.StatusBadRequest
	}
	return err
}
