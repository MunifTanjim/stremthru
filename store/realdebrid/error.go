package realdebrid

import (
	"net/http"

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

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameRealDebrid)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Err
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
	return err
}
