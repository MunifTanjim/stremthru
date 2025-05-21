package easydebrid

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNameEasyDebrid)

	if rerr, ok := cause.(*ResponseContainer); ok {
		err.Msg = rerr.Err
		if err.Msg == "Account not premium." {
			err.Code = core.ErrorCodePaymentRequired
			err.StatusCode = http.StatusPaymentRequired
		}
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
