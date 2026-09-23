package webtor

import (
	"errors"
	"strconv"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
	webtorsdk "github.com/webtor-io/api-sdk-go"
)

var errorCodeByCode = map[string]core.ErrorCode{
	webtorsdk.CodeBadRequest:      core.ErrorCodeBadRequest,
	webtorsdk.CodeUnauthorized:    core.ErrorCodeUnauthorized,
	webtorsdk.CodeForbidden:       core.ErrorCodeForbidden,
	webtorsdk.CodePaymentRequired: core.ErrorCodePaymentRequired,
	webtorsdk.CodeNotFound:        core.ErrorCodeNotFound,
	webtorsdk.CodeConflict:        core.ErrorCodeConflict,
	webtorsdk.CodeRateLimited:     core.ErrorCodeTooManyRequests,
	webtorsdk.CodeUnavailable:     core.ErrorCodeServiceUnavailable,
	webtorsdk.CodeInternal:        core.ErrorCodeInternalServerError,
	webtorsdk.CodeUpstream:        core.ErrorCodeBadGateway,
	webtorsdk.CodeUpstreamTimeout: core.ErrorCodeBadGateway,
}

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameWebtor)

	var aerr *webtorsdk.Error
	if errors.As(cause, &aerr) {
		err.Msg = aerr.Message
		if err.Msg == "" {
			err.Msg = aerr.Code
		}
		if code, found := errorCodeByCode[aerr.Code]; found {
			err.Code = code
		}
		err.StatusCode = aerr.HTTPStatus
		if aerr.RetryAfter > 0 {
			err.RetryAfter = strconv.Itoa(int(aerr.RetryAfter.Seconds()))
		}
		err.UpstreamCause = aerr
	} else {
		err.Cause = cause
	}

	return err
}
