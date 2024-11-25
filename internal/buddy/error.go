package buddy

import (
	"github.com/MunifTanjim/stremthru/core"
)

type ErrorCode = core.ErrorCode

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Message
		err.Code = rerr.Code
		err.StatusCode = rerr.StatusCode
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
