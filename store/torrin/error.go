package torrin

import (
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameTorrin)

	if rerr, ok := cause.(*ResponseError); ok {
		err.Msg = rerr.Message
		err.StatusCode = rerr.Code
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
