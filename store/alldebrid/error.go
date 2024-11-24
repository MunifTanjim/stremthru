package alldebrid

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type ErrorCode = core.ErrorCode

const (
	ErrorCodeAuthMissingAgent  ErrorCode = "AUTH_MISSING_AGENT"
	ErrorCodeAuthBadAgent      ErrorCode = "AUTH_BAD_AGENT"
	ErrorCodeAuthMissingAPIKey ErrorCode = "AUTH_MISSING_APIKEY"
	ErrorCodeAuthBadAPIKey     ErrorCode = "AUTH_BAD_APIKEY"
	ErrorCodeAuthBlocked       ErrorCode = "AUTH_BLOCKED"
	ErrorCodeAuthBanned        ErrorCode = "AUTH_USER_BANNED"
)

var StatusCodeByErrorCode = map[ErrorCode]int{
	ErrorCodeAuthMissingAgent:  http.StatusBadRequest,
	ErrorCodeAuthMissingAPIKey: http.StatusUnauthorized,
	ErrorCodeAuthBadAPIKey:     http.StatusUnauthorized,
	ErrorCodeAuthBlocked:       http.StatusForbidden,
	ErrorCodeAuthBanned:        http.StatusForbidden,
}

type MagnetErrorCode = core.ErrorCode

const (
	MagnetErrorCodeNoURI         MagnetErrorCode = "MAGNET_NO_URI"
	MagnetErrorCodeInvalidId     MagnetErrorCode = "MAGNET_INVALID_ID"
	MagnetErrorCodeInvalidURI    MagnetErrorCode = "MAGNET_INVALID_URI"
	MagnetErrorCodeMustBePremium MagnetErrorCode = "MAGNET_MUST_BE_PREMIUM"
	MagnetErrorCodeNoServer      MagnetErrorCode = "MAGNET_NO_SERVER"
	MagnetErrorCodeTooManyActive MagnetErrorCode = "MAGNET_TOO_MANY_ACTIVE"
	MagnetErrorCodeProcessing    MagnetErrorCode = "MAGNET_PROCESSING" // for RestartMagnet
)

var StatusCodeByMagnetErrorCode = map[MagnetErrorCode]int{
	MagnetErrorCodeNoURI:         http.StatusBadRequest,
	MagnetErrorCodeInvalidId:     http.StatusBadRequest,
	MagnetErrorCodeInvalidURI:    http.StatusBadRequest,
	MagnetErrorCodeMustBePremium: http.StatusPaymentRequired,
	MagnetErrorCodeNoServer:      http.StatusUnprocessableEntity,
	MagnetErrorCodeTooManyActive: http.StatusUnprocessableEntity,
}

type LinkErrorCode = core.ErrorCode

const (
	LinkErrorCodeHostNotSupported     LinkErrorCode = "LINK_HOST_NOT_SUPPORTED"
	LinkErrorCodeDown                 LinkErrorCode = "LINK_DOWN"
	LinkErrorCodeHostUnavailable      LinkErrorCode = "LINK_HOST_UNAVAILABLE"
	LinkErrorCodeTooManyDownloads     LinkErrorCode = "LINK_TOO_MANY_DOWNLOADS"
	LinkErrorCodeHostFull             LinkErrorCode = "LINK_HOST_FULL"
	LinkErrorCodeHostLimitReached     LinkErrorCode = "LINK_HOST_LIMIT_REACHED"
	LinkErrorCodePassProtected        LinkErrorCode = "LINK_PASS_PROTECTED"
	LinkErrorCodeError                LinkErrorCode = "LINK_ERROR"
	LinkErrorCodeNotSupported         LinkErrorCode = "LINK_NOT_SUPPORTED"
	LinkErrorCodeTemporaryUnavailable LinkErrorCode = "LINK_TEMPORARY_UNAVAILABLE"
	MustBePremium                     LinkErrorCode = "MUST_BE_PREMIUM"
	FreeTrialLimitReached             LinkErrorCode = "FREE_TRIAL_LIMIT_REACHED"
	NoServer                          LinkErrorCode = "NO_SERVER"
)

var StatusCodeByLinkErrorCode = map[LinkErrorCode]int{
	LinkErrorCodeHostNotSupported:     http.StatusNotImplemented,
	LinkErrorCodeDown:                 http.StatusServiceUnavailable,
	LinkErrorCodeHostUnavailable:      http.StatusServiceUnavailable,
	LinkErrorCodeTooManyDownloads:     http.StatusTooManyRequests,
	LinkErrorCodeHostFull:             http.StatusUnprocessableEntity,
	LinkErrorCodeHostLimitReached:     http.StatusUnprocessableEntity,
	LinkErrorCodePassProtected:        http.StatusForbidden,
	LinkErrorCodeError:                http.StatusInternalServerError,
	LinkErrorCodeNotSupported:         http.StatusNotImplemented,
	LinkErrorCodeTemporaryUnavailable: http.StatusServiceUnavailable,
	MustBePremium:                     http.StatusPaymentRequired,
	FreeTrialLimitReached:             http.StatusPaymentRequired,
	NoServer:                          http.StatusServiceUnavailable,
}

var errorCodeByErrorCode = map[ErrorCode]core.ErrorCode{
	ErrorCodeAuthMissingAgent:  core.ErrorCodeBadRequest,
	ErrorCodeAuthBadAgent:      core.ErrorCodeBadRequest,
	ErrorCodeAuthMissingAPIKey: core.ErrorCodeUnauthorized,
	ErrorCodeAuthBadAPIKey:     core.ErrorCodeUnauthorized,
	ErrorCodeAuthBlocked:       core.ErrorCodeForbidden,
	ErrorCodeAuthBanned:        core.ErrorCodeForbidden,

	MagnetErrorCodeNoURI:         core.ErrorCodeMagnetInvalid,
	MagnetErrorCodeInvalidId:     core.ErrorCodeMagnetInvalid,
	MagnetErrorCodeInvalidURI:    core.ErrorCodeMagnetInvalid,
	MagnetErrorCodeMustBePremium: core.ErrorCodePaymentRequired,
	MagnetErrorCodeNoServer:      core.ErrorCodeForbidden,
	MagnetErrorCodeTooManyActive: core.ErrorCodeStoreLimitExceeded,
	MagnetErrorCodeProcessing:    core.ErrorCodeConflict,

	LinkErrorCodeHostNotSupported:     core.ErrorCodeNotImplemented,
	LinkErrorCodeDown:                 core.ErrorCodeServiceUnavailable,
	LinkErrorCodeHostUnavailable:      core.ErrorCodeServiceUnavailable,
	LinkErrorCodeTooManyDownloads:     core.ErrorCodeStoreLimitExceeded,
	LinkErrorCodeHostFull:             core.ErrorCodeStoreLimitExceeded,
	LinkErrorCodeHostLimitReached:     core.ErrorCodeStoreLimitExceeded,
	LinkErrorCodePassProtected:        core.ErrorCodeForbidden,
	LinkErrorCodeError:                core.ErrorCodeInternalServerError,
	LinkErrorCodeNotSupported:         core.ErrorCodeNotImplemented,
	LinkErrorCodeTemporaryUnavailable: core.ErrorCodeServiceUnavailable,
	MustBePremium:                     core.ErrorCodePaymentRequired,
	FreeTrialLimitReached:             core.ErrorCodeStoreLimitExceeded,
	NoServer:                          core.ErrorCodeServiceUnavailable,
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
		if sc := StatusCodeByErrorCode[rerr.Code]; sc != 0 {
			err.StatusCode = sc
		}
		err.UpstreamCause = rerr
	} else if merr, ok := cause.(*MagnetError); ok {
		err.Msg = merr.Message
		err.Code = TranslateErrorCode(merr.Code)
		if sc := StatusCodeByMagnetErrorCode[merr.Code]; sc != 0 {
			err.StatusCode = sc
		}
		err.UpstreamCause = merr
	} else {
		err.Cause = cause
	}

	return err
}
