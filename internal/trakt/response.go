package trakt

import (
	"encoding/json"

	"github.com/MunifTanjim/stremthru/core"
)

type paginatedResponseData[T any] struct {
	ResponseError
	data []T
}

func (d *paginatedResponseData[T]) UnmarshalJSON(data []byte) error {
	var rerr ResponseError
	if err := json.Unmarshal(data, &rerr); err == nil && rerr.Err != "" {
		d.ResponseError = rerr
		return nil
	}

	var items []T
	err := json.Unmarshal(data, &items)
	if err == nil {
		d.data = items
		return nil
	}

	e := core.NewAPIError("failed to parse response")
	e.Cause = err
	return e
}
