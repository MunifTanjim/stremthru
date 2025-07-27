package debrider

import (
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameDebrider)

	if rerr, ok := cause.(*ResponseContainer); ok {
		err.Msg = rerr.Message
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
