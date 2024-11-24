package debridlink

import (
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

var errorCodeByErrorCode = map[ErrorCode]core.ErrorCode{
	ErrorCodeBadToken:                core.ErrorCodeUnauthorized,
	ErrorCodeBadSign:                 core.ErrorCodeUnauthorized,
	ErrorCodeHidedToken:              core.ErrorCodeUnauthorized,
	ErrorCodeServerError:             core.ErrorCodeInternalServerError,
	ErrorCodeAccessDenied:            core.ErrorCodeUnauthorized,
	ErrorCodeAuthorizationPending:    core.ErrorCodeUnauthorized,
	ErrorCodeUnsupportedGrantType:    core.ErrorCodeBadRequest,
	ErrorCodeUnsupportedResponseType: core.ErrorCodeUnsupportedMediaType,
	ErrorCodeInvalidRequest:          core.ErrorCodeBadRequest,
	ErrorCodeInvalidScope:            core.ErrorCodeBadRequest,
	ErrorCodeExpiredToken:            core.ErrorCodeUnauthorized,
	ErrorCodeUnauthorizedClient:      core.ErrorCodeUnauthorized,
	ErrorCodeInvalidClient:           core.ErrorCodeUnauthorized,
	ErrorCodeUnknowR:                 core.ErrorCodeInternalServerError,
	ErrorCodeInternalError:           core.ErrorCodeInternalServerError,
	ErrorCodeBadArguments:            core.ErrorCodeBadRequest,
	ErrorCodeBadId:                   core.ErrorCodeBadRequest,
	ErrorCodeFloodDetected:           core.ErrorCodeTooManyRequests,
	ErrorCodeServerNotAllowed:        core.ErrorCodeForbidden,
	ErrorCodeFreeServerOverload:      core.ErrorCodeServiceUnavailable,
	ErrorCodeMaxAttempts:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeCaptchaRequired:         core.ErrorCodeForbidden,
	ErrorCodeAccountLocked:           core.ErrorCodeForbidden,
	ErrorCodeNotDebrid:               core.ErrorCodeBadRequest,
	ErrorCodeHostNotValid:            core.ErrorCodeBadRequest,
	ErrorCodeFileNotFound:            core.ErrorCodeNotFound,
	ErrorCodeFileNotAvailable:        core.ErrorCodeNotFound,
	ErrorCodeBadFileUrl:              core.ErrorCodeBadRequest,
	ErrorCodeBadFilePassword:         core.ErrorCodeUnauthorized,
	ErrorCodeNotFreeHost:             core.ErrorCodePaymentRequired,
	ErrorCodeMaintenanceHost:         core.ErrorCodeServiceUnavailable,
	ErrorCodeNoServerHost:            core.ErrorCodeServiceUnavailable,
	ErrorCodeMaxLink:                 core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMaxLinkHost:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMaxData:                 core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMaxDataHost:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeDisabledServerHost:      core.ErrorCodeServiceUnavailable,
	ErrorCodeNotAddTorrent:           core.ErrorCodeBadRequest,
	ErrorCodeTorrentTooBig:           core.ErrorCodeUnprocessableEntity,
	ErrorCodeMaxTorrent:              core.ErrorCodeStoreLimitExceeded,
}

func TranslateErrorCode(errorCode ErrorCode) core.ErrorCode {
	if code, found := errorCodeByErrorCode[errorCode]; found {
		return code
	}
	return core.ErrorCodeUnknown
}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameDebridLink)

	if rerr, ok := cause.(*ResponseError); ok {
		if rerr.ErrDesc != "" {
			err.Msg = rerr.ErrDesc
		} else {
			err.Msg = "Debrid-Link Error Code: " + string(rerr.Err)
		}
		err.Code = TranslateErrorCode(rerr.Err)
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
