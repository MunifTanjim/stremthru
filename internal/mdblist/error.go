package mdblist

import "github.com/MunifTanjim/stremthru/core"

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")

	if rerr, ok := cause.(*ResponseContainer); ok {
		err.Msg = rerr.Err
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
