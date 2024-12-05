package store

import "github.com/MunifTanjim/stremthru/core"

var ErrorInvalidStoreName = func(name string) *core.StoreError {
	err := core.NewStoreError("invalid store name")
	err.Code = core.ErrorCodeStoreNameInvalid
	err.StoreName = name
	return err
}
