package alldebrid

import (
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type ErrorCode = core.ErrorCode

const (
	ErrorCodeGeneric     ErrorCode = "GENERIC"
	ErrorCode404         ErrorCode = "404"
	ErrorCodeMaintenance ErrorCode = "MAINTENANCE"

	ErrorCodeAuthMissingAPIKey ErrorCode = "AUTH_MISSING_APIKEY"
	ErrorCodeAuthBadAPIKey     ErrorCode = "AUTH_BAD_APIKEY"
	ErrorCodeAuthBlocked       ErrorCode = "AUTH_BLOCKED"
	ErrorCodeAuthBanned        ErrorCode = "AUTH_USER_BANNED"

	ErrorCodeAlreadySent ErrorCode = "ALREADY_SENT"
	ErrorCodeNoServer    ErrorCode = "NO_SERVER"

	ErrorCodeLinkIsMissing            ErrorCode = "LINK_IS_MISSING"
	ErrorCodeBadLink                  ErrorCode = "BAD_LINK"
	ErrorCodeLinkHostNotSupported     ErrorCode = "LINK_HOST_NOT_SUPPORTED"
	ErrorCodeLinkDown                 ErrorCode = "LINK_DOWN"
	ErrorCodeLinkPassProtected        ErrorCode = "LINK_PASS_PROTECTED"
	ErrorCodeLinkHostUnavailable      ErrorCode = "LINK_HOST_UNAVAILABLE"
	ErrorCodeLinkTooManyDownloads     ErrorCode = "LINK_TOO_MANY_DOWNLOADS"
	ErrorCodeLinkHostFull             ErrorCode = "LINK_HOST_FULL"
	ErrorCodeLinkHostLimitReached     ErrorCode = "LINK_HOST_LIMIT_REACHED"
	ErrorCodeLinkError                ErrorCode = "LINK_ERROR"
	ErrorCodeLinkTemporaryUnavailable ErrorCode = "LINK_TEMPORARY_UNAVAILABLE"
	ErrorCodeLinkNotSupported         ErrorCode = "LINK_NOT_SUPPORTED"

	ErrorCodeRedirectorNotSupported ErrorCode = "REDIRECTOR_NOT_SUPPORTED"
	ErrorCodeRedirectorError        ErrorCode = "REDIRECTOR_ERROR"
	ErrorCodeStreamInvalidGenID     ErrorCode = "STREAM_INVALID_GEN_ID"
	ErrorCodeStreamInvalidStreamID  ErrorCode = "STREAM_INVALID_STREAM_ID"
	ErrorCodeDelayedInvalidID       ErrorCode = "DELAYED_INVALID_ID"

	ErrorCodeFreeTrialLimitReached ErrorCode = "FREE_TRIAL_LIMIT_REACHED"
	ErrorCodeMustBePremium         ErrorCode = "MUST_BE_PREMIUM"

	ErrorCodeMagnetInvalidId        ErrorCode = "MAGNET_INVALID_ID"
	ErrorCodeMagnetInvalidURI       ErrorCode = "MAGNET_INVALID_URI"
	ErrorCodeMagnetInvalidFile      ErrorCode = "MAGNET_INVALID_FILE"
	ErrorCodeMagnetFileUploadFailed ErrorCode = "MAGNET_FILE_UPLOAD_FAILED"
	ErrorCodeMagnetNoURI            ErrorCode = "MAGNET_NO_URI"
	ErrorCodeMagnetProcessing       ErrorCode = "MAGNET_PROCESSING" // for RestartMagnet
	ErrorCodeMagnetTooManyActive    ErrorCode = "MAGNET_TOO_MANY_ACTIVE"
	ErrorCodeMagnetTooMany          ErrorCode = "MAGNET_TOO_MANY"
	ErrorCodeMagnetMustBePremium    ErrorCode = "MAGNET_MUST_BE_PREMIUM"
	ErrorCodeMagnetTooLarge         ErrorCode = "MAGNET_TOO_LARGE"
	ErrorCodeMagnetUploadFailed     ErrorCode = "MAGNET_UPLOAD_FAILED"
	ErrorCodeMagnetInternalError    ErrorCode = "MAGNET_INTERNAL_ERROR"
	ErrorCodeMagnetCantBootstrap    ErrorCode = "MAGNET_CANT_BOOTSTRAP"
	ErrorCodeMagnetTooBig           ErrorCode = "MAGNET_MAGNET_TOO_BIG"
	ErrorCodeMagnetTookTooLong      ErrorCode = "MAGNET_TOOK_TOO_LONG"
	ErrorCodeMagnetLinksRemoved     ErrorCode = "MAGNET_LINKS_REMOVED"
	ErrorCodeMagnetProcessingFailed ErrorCode = "MAGNET_PROCESSING_FAILED"
	ErrorCodeMagnetNoServer         ErrorCode = "MAGNET_NO_SERVER"
	// undocumented
	ErrorCodeMagnetProcessingCooldown ErrorCode = "MAGNET_PROCESSING_COOLDOWN"

	ErrorCodePinAlreadyAuthed ErrorCode = "PIN_ALREADY_AUTHED"
	ErrorCodePinExpired       ErrorCode = "PIN_EXPIRED"
	ErrorCodePinInvalid       ErrorCode = "PIN_INVALID"

	ErrorCodeUserLinkMissing ErrorCode = "USER_LINK_MISSING"
	ErrorCodeUserLinkInvalid ErrorCode = "USER_LINK_INVALID"

	ErrorCodeMissingNotifEndpoint ErrorCode = "MISSING_NOTIF_ENDPOINT"

	ErrorCodeVoucherDurationInvalid ErrorCode = "VOUCHER_DURATION_INVALID"
	ErrorCodeVoucherNbInvalid       ErrorCode = "VOUCHER_NB_INVALID"
	ErrorCodeNoMoreVoucher          ErrorCode = "NO_MORE_VOUCHER"

	ErrorCodeInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	ErrorCodeDownloadFailed      ErrorCode = "DOWNLOAD_FAILED"
	ErrorCodeAccountInvalid      ErrorCode = "ACCOUNT_INVALID"
	ErrorCodeNoJSONParam         ErrorCode = "NO_JSON_PARAM"
	ErrorCodeJSONInvalid         ErrorCode = "JSON_INVALID"

	ErrorCodeFreedaysInvalidCountry  ErrorCode = "FREEDAYS_INVALID_COUNTRY"
	ErrorCodeFreedaysInvalidPhone    ErrorCode = "FREEDAYS_INVALID_PHONE"
	ErrorCodeFreedaysInvalidProvider ErrorCode = "FREEDAYS_INVALID_PROVIDER"

	ErrorCodeStreamInvalidGenId    ErrorCode = "STREAM_INVALID_GEN_ID"
	ErrorCodeStreamInvalidStreamId ErrorCode = "STREAM_INVALID_STREAM_ID"
	ErrorCodeDelayedInvalidId      ErrorCode = "DELAYED_INVALID_ID"
)

var errorCodeByErrorCode = map[ErrorCode]core.ErrorCode{
	ErrorCodeGeneric:     core.ErrorCodeUnknown,
	ErrorCode404:         core.ErrorCodeNotFound,
	ErrorCodeMaintenance: core.ErrorCodeServiceUnavailable,

	ErrorCodeAuthMissingAPIKey: core.ErrorCodeUnauthorized,
	ErrorCodeAuthBadAPIKey:     core.ErrorCodeUnauthorized,
	ErrorCodeAuthBlocked:       core.ErrorCodeForbidden,
	ErrorCodeAuthBanned:        core.ErrorCodeForbidden,

	ErrorCodeAlreadySent: core.ErrorCodeUnknown,
	ErrorCodeNoServer:    core.ErrorCodeForbidden,

	ErrorCodeLinkIsMissing:            core.ErrorCodeUnknown,
	ErrorCodeBadLink:                  core.ErrorCodeBadRequest,
	ErrorCodeLinkHostNotSupported:     core.ErrorCodeNotImplemented,
	ErrorCodeLinkDown:                 core.ErrorCodeServiceUnavailable,
	ErrorCodeLinkPassProtected:        core.ErrorCodeForbidden,
	ErrorCodeLinkHostUnavailable:      core.ErrorCodeServiceUnavailable,
	ErrorCodeLinkTooManyDownloads:     core.ErrorCodeStoreLimitExceeded,
	ErrorCodeLinkHostFull:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeLinkHostLimitReached:     core.ErrorCodeStoreLimitExceeded,
	ErrorCodeLinkError:                core.ErrorCodeInternalServerError,
	ErrorCodeLinkTemporaryUnavailable: core.ErrorCodeServiceUnavailable,
	ErrorCodeLinkNotSupported:         core.ErrorCodeNotImplemented,

	ErrorCodeRedirectorNotSupported: core.ErrorCodeNotImplemented,
	ErrorCodeRedirectorError:        core.ErrorCodeInternalServerError,
	ErrorCodeStreamInvalidGenID:     core.ErrorCodeBadRequest,
	ErrorCodeStreamInvalidStreamID:  core.ErrorCodeBadRequest,
	ErrorCodeDelayedInvalidID:       core.ErrorCodeBadRequest,

	ErrorCodeFreeTrialLimitReached: core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMustBePremium:         core.ErrorCodePaymentRequired,

	ErrorCodeMagnetInvalidId:        core.ErrorCodeStoreMagnetInvalid,
	ErrorCodeMagnetInvalidURI:       core.ErrorCodeStoreMagnetInvalid,
	ErrorCodeMagnetInvalidFile:      core.ErrorCodeStoreMagnetInvalid,
	ErrorCodeMagnetFileUploadFailed: core.ErrorCodeInternalServerError,
	ErrorCodeMagnetNoURI:            core.ErrorCodeStoreMagnetInvalid,
	ErrorCodeMagnetProcessing:       core.ErrorCodeConflict,
	ErrorCodeMagnetTooManyActive:    core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMagnetTooMany:          core.ErrorCodeStoreLimitExceeded,
	ErrorCodeMagnetMustBePremium:    core.ErrorCodePaymentRequired,
	ErrorCodeMagnetTooLarge:         core.ErrorCodeBadRequest,
	ErrorCodeMagnetUploadFailed:     core.ErrorCodeInternalServerError,
	ErrorCodeMagnetInternalError:    core.ErrorCodeInternalServerError,
	ErrorCodeMagnetCantBootstrap:    core.ErrorCodeUnprocessableEntity,
	ErrorCodeMagnetTooBig:           core.ErrorCodeBadRequest,
	ErrorCodeMagnetTookTooLong:      core.ErrorCodeUnprocessableEntity,
	ErrorCodeMagnetLinksRemoved:     core.ErrorCodeNotFound,
	ErrorCodeMagnetProcessingFailed: core.ErrorCodeUnprocessableEntity,
	ErrorCodeMagnetNoServer:         core.ErrorCodeForbidden,

	ErrorCodePinAlreadyAuthed: core.ErrorCodeUnknown,
	ErrorCodePinExpired:       core.ErrorCodeBadRequest,
	ErrorCodePinInvalid:       core.ErrorCodeBadRequest,

	ErrorCodeUserLinkMissing: core.ErrorCodeBadRequest,
	ErrorCodeUserLinkInvalid: core.ErrorCodeBadRequest,

	ErrorCodeMissingNotifEndpoint: core.ErrorCodeBadRequest,

	ErrorCodeVoucherDurationInvalid: core.ErrorCodeBadRequest,
	ErrorCodeVoucherNbInvalid:       core.ErrorCodeBadRequest,
	ErrorCodeNoMoreVoucher:          core.ErrorCodeBadRequest,

	ErrorCodeInsufficientBalance: core.ErrorCodePaymentRequired,
	ErrorCodeDownloadFailed:      core.ErrorCodeInternalServerError,
	ErrorCodeAccountInvalid:      core.ErrorCodeForbidden,
	ErrorCodeNoJSONParam:         core.ErrorCodeBadRequest,
	ErrorCodeJSONInvalid:         core.ErrorCodeBadRequest,

	ErrorCodeFreedaysInvalidCountry:  core.ErrorCodeBadRequest,
	ErrorCodeFreedaysInvalidPhone:    core.ErrorCodeBadRequest,
	ErrorCodeFreedaysInvalidProvider: core.ErrorCodeBadRequest,
}

func TranslateErrorCode(errorCode ErrorCode) core.ErrorCode {
	if code, found := errorCodeByErrorCode[errorCode]; found {
		return code
	}
	return core.ErrorCodeUnknown
}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameAlldebrid)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Message
		err.Code = TranslateErrorCode(rerr.Code)
		err.UpstreamCause = rerr
	} else if merr, ok := cause.(*MagnetError); ok {
		err.Msg = merr.Message
		err.Code = TranslateErrorCode(merr.Code)
		err.UpstreamCause = merr
	} else {
		err.Cause = cause
	}

	return err
}
