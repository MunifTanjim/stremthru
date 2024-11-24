package realdebrid

import (
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type ErrorCode int

const (
	ErrorCodeInternal         = -1
	ErrorCodeMissingParameter = iota
	ErrorCodeBadParameterValue
	ErrorCodeUnknownMethod
	ErrorCodeMethodNotAllowed
	ErrorCodeSlowDown
	ErrorCodeResourceUnreachable
	ErrorCodeResourceNotFound
	ErrorCodeBadToken
	ErrorCodePermissionDenied
	ErrorCodeTwoFactorAuthenticationNeeded
	ErrorCodeTwoFactorAuthenticationPending
	ErrorCodeInvalidLogin
	ErrorCodeInvalidPassword
	ErrorCodeAccountLocked
	ErrorCodeAccountNotActivated
	ErrorCodeUnsupportedHoster
	ErrorCodeHosterInMaintenance
	ErrorCodeHosterLimitReached
	ErrorCodeHosterTemporarilyUnavailable
	ErrorCodeHosterNotAvailableForFreeUsers
	ErrorCodeTooManyActiveDownloads
	ErrorCodeIPAddressNotAllowed
	ErrorCodeTrafficExhausted
	ErrorCodeFileUnavailable
	ErrorCodeServiceUnavailable
	ErrorCodeUploadTooBig
	ErrorCodeUploadError
	ErrorCodeFileNotAllowed
	ErrorCodeTorrentTooBig
	ErrorCodeTorrentFileInvalid
	ErrorCodeActionAlreadyDone
	ErrorCodeImageResolutionError
	ErrorCodeTorrentAlreadyActive
	ErrorCodeTooManyRequests
	ErrorCodeInfringingFile
	ErrorCodeFairUsageLimit
)

var errorCodeByErrorCode = map[ErrorCode]core.ErrorCode{
	ErrorCodeInternal:                       core.ErrorCodeInternalServerError,
	ErrorCodeMissingParameter:               core.ErrorCodeBadRequest,
	ErrorCodeBadParameterValue:              core.ErrorCodeBadRequest,
	ErrorCodeUnknownMethod:                  core.ErrorCodeMethodNotAllowed,
	ErrorCodeMethodNotAllowed:               core.ErrorCodeMethodNotAllowed,
	ErrorCodeSlowDown:                       core.ErrorCodeTooManyRequests,
	ErrorCodeResourceUnreachable:            core.ErrorCodeBadGateway,
	ErrorCodeResourceNotFound:               core.ErrorCodeNotFound,
	ErrorCodeBadToken:                       core.ErrorCodeUnauthorized,
	ErrorCodePermissionDenied:               core.ErrorCodeForbidden,
	ErrorCodeTwoFactorAuthenticationNeeded:  core.ErrorCodeUnauthorized,
	ErrorCodeTwoFactorAuthenticationPending: core.ErrorCodeUnauthorized,
	ErrorCodeInvalidLogin:                   core.ErrorCodeUnauthorized,
	ErrorCodeInvalidPassword:                core.ErrorCodeUnauthorized,
	ErrorCodeAccountLocked:                  core.ErrorCodeForbidden,
	ErrorCodeAccountNotActivated:            core.ErrorCodeForbidden,
	ErrorCodeUnsupportedHoster:              core.ErrorCodeBadGateway,
	ErrorCodeHosterInMaintenance:            core.ErrorCodeServiceUnavailable,
	ErrorCodeHosterLimitReached:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeHosterTemporarilyUnavailable:   core.ErrorCodeServiceUnavailable,
	ErrorCodeHosterNotAvailableForFreeUsers: core.ErrorCodePaymentRequired,
	ErrorCodeTooManyActiveDownloads:         core.ErrorCodeStoreLimitExceeded,
	ErrorCodeIPAddressNotAllowed:            core.ErrorCodeForbidden,
	ErrorCodeTrafficExhausted:               core.ErrorCodeStoreLimitExceeded,
	ErrorCodeFileUnavailable:                core.ErrorCodeNotFound,
	ErrorCodeServiceUnavailable:             core.ErrorCodeServiceUnavailable,
	ErrorCodeUploadTooBig:                   core.ErrorCodeUnprocessableEntity,
	ErrorCodeUploadError:                    core.ErrorCodeUnknown,
	ErrorCodeFileNotAllowed:                 core.ErrorCodeUnprocessableEntity,
	ErrorCodeTorrentTooBig:                  core.ErrorCodeUnprocessableEntity,
	ErrorCodeTorrentFileInvalid:             core.ErrorCodeBadRequest,
	ErrorCodeActionAlreadyDone:              core.ErrorCodeConflict,
	ErrorCodeImageResolutionError:           core.ErrorCodeBadRequest,
	ErrorCodeTorrentAlreadyActive:           core.ErrorCodeConflict,
	ErrorCodeTooManyRequests:                core.ErrorCodeTooManyRequests,
	ErrorCodeInfringingFile:                 core.ErrorCodeUnavailableForLegalReasons,
	ErrorCodeFairUsageLimit:                 core.ErrorCodeStoreLimitExceeded,
}

func TranslateErrorCode(errorCode ErrorCode) core.ErrorCode {
	if code, found := errorCodeByErrorCode[errorCode]; found {
		return code
	}
	return core.ErrorCodeUnknown
}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameRealDebrid)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Err
		err.Code = TranslateErrorCode(rerr.ErrCode)
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
