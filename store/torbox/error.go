package torbox

import (
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

type ErrorCode string

const (
	ErrorCodeDatabaseError           ErrorCode = "DATABASE_ERROR"
	ErrorCodeUnknownError            ErrorCode = "UNKNOWN_ERROR"
	ErrorCodeNoAuth                  ErrorCode = "NO_AUTH"
	ErrorCodeBadToken                ErrorCode = "BAD_TOKEN"
	ErrorCodeAuthError               ErrorCode = "AUTH_ERROR"
	ErrorCodeInvalidOption           ErrorCode = "INVALID_OPTION"
	ErrorCodeRedirectError           ErrorCode = "REDIRECT_ERROR"
	ErrorCodeOAuthVerificationError  ErrorCode = "OAUTH_VERIFICATION_ERROR"
	ErrorCodeEndpointNotFound        ErrorCode = "ENDPOINT_NOT_FOUND"
	ErrorCodeItemNotFound            ErrorCode = "ITEM_NOT_FOUND"
	ErrorCodePlanRestrictedFeature   ErrorCode = "PLAN_RESTRICTED_FEATURE"
	ErrorCodeDuplicateItem           ErrorCode = "DUPLICATE_ITEM"
	ErrorCodeBozoRssFeed             ErrorCode = "BOZO_RSS_FEED"
	ErrorCodeSellixError             ErrorCode = "SELLIX_ERROR"
	ErrorCodeTooMuchData             ErrorCode = "TOO_MUCH_DATA"
	ErrorCodeDownloadTooLarge        ErrorCode = "DOWNLOAD_TOO_LARGE"
	ErrorCodeMissingRequiredOption   ErrorCode = "MISSING_REQUIRED_OPTION"
	ErrorCodeTooManyOptions          ErrorCode = "TOO_MANY_OPTIONS"
	ErrorCodeBozoTorrent             ErrorCode = "BOZO_TORRENT"
	ErrorCodeNoServersAvailableError ErrorCode = "NO_SERVERS_AVAILABLE_ERROR"
	ErrorCodeMonthlyLimit            ErrorCode = "MONTHLY_LIMIT"
	ErrorCodeCooldownLimit           ErrorCode = "COOLDOWN_LIMIT"
	ErrorCodeActiveLimit             ErrorCode = "ACTIVE_LIMIT"
	ErrorCodeDownloadServerError     ErrorCode = "DOWNLOAD_SERVER_ERROR"
	ErrorCodeBozoNzb                 ErrorCode = "BOZO_NZB"
	ErrorCodeSearchError             ErrorCode = "SEARCH_ERROR"
	ErrorCodeInvalidDevice           ErrorCode = "INVALID_DEVICE"
	ErrorCodeDiffIssue               ErrorCode = "DIFF_ISSUE"
	ErrorCodeLinkOffline             ErrorCode = "LINK_OFFLINE"
	ErrorCodeVendorDisabled          ErrorCode = "VENDOR_DISABLED"
)

var errorCodeByErrorCode = map[ErrorCode]core.ErrorCode{
	ErrorCodeDatabaseError:           core.ErrorCodeInternalServerError,
	ErrorCodeUnknownError:            core.ErrorCodeInternalServerError,
	ErrorCodeNoAuth:                  core.ErrorCodeUnauthorized,
	ErrorCodeBadToken:                core.ErrorCodeUnauthorized,
	ErrorCodeAuthError:               core.ErrorCodeUnauthorized,
	ErrorCodeInvalidOption:           core.ErrorCodeBadRequest,
	ErrorCodeRedirectError:           core.ErrorCodeInternalServerError,
	ErrorCodeOAuthVerificationError:  core.ErrorCodeUnauthorized,
	ErrorCodeEndpointNotFound:        core.ErrorCodeNotFound,
	ErrorCodeItemNotFound:            core.ErrorCodeNotFound,
	ErrorCodePlanRestrictedFeature:   core.ErrorCodePaymentRequired,
	ErrorCodeDuplicateItem:           core.ErrorCodeConflict,
	ErrorCodeBozoRssFeed:             core.ErrorCodeBadRequest,
	ErrorCodeSellixError:             core.ErrorCodeInternalServerError,
	ErrorCodeTooMuchData:             core.ErrorCodeUnprocessableEntity,
	ErrorCodeDownloadTooLarge:        core.ErrorCodeUnprocessableEntity,
	ErrorCodeMissingRequiredOption:   core.ErrorCodeBadRequest,
	ErrorCodeTooManyOptions:          core.ErrorCodeBadRequest,
	ErrorCodeBozoTorrent:             core.ErrorCodeBadRequest,
	ErrorCodeNoServersAvailableError: core.ErrorCodeServiceUnavailable,
	ErrorCodeMonthlyLimit:            core.ErrorCodeStoreLimitExceeded,
	ErrorCodeCooldownLimit:           core.ErrorCodeTooManyRequests,
	ErrorCodeActiveLimit:             core.ErrorCodeStoreLimitExceeded,
	ErrorCodeDownloadServerError:     core.ErrorCodeInternalServerError,
	ErrorCodeBozoNzb:                 core.ErrorCodeBadRequest,
	ErrorCodeSearchError:             core.ErrorCodeInternalServerError,
	ErrorCodeInvalidDevice:           core.ErrorCodeBadRequest,
	ErrorCodeDiffIssue:               core.ErrorCodeUnknown,
	ErrorCodeLinkOffline:             core.ErrorCodeServiceUnavailable,
	ErrorCodeVendorDisabled:          core.ErrorCodeServiceUnavailable,
}

func TranslateErrorCode(errorCode ErrorCode) core.ErrorCode {
	if code, found := errorCodeByErrorCode[errorCode]; found {
		return code
	}
	return core.ErrorCodeUnknown

}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameTorBox)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Detail
		err.Code = TranslateErrorCode(rerr.Err)
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
