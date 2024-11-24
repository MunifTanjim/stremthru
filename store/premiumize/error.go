package premiumize

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
)

func UpstreamErrorWithCause(cause error) *core.UpstreamError {
	err := core.NewUpstreamError("")
	err.StoreName = string(store.StoreNamePremiumize)

	if rerr, ok := cause.(*ResponseContainer); ok {
		err.Msg = rerr.Message
		if err.Msg == "Not logged in." {
			err.Code = core.ErrorCodeUnauthorized
			err.StatusCode = http.StatusUnauthorized
		}
		err.UpstreamCause = rerr
	} else {
		err.Cause = cause
	}

	return err
}
