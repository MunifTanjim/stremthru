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
	if err.StatusCode <= http.StatusBadRequest {
		err.StatusCode = http.StatusBadRequest
	}
	return err
}
