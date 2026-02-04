package dash_api

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/server"
)

var SendData = server.SendData
var SendError = server.SendError

func ReadRequestBodyJSON[T any](r *http.Request, payload T) error {
	return server.ReadRequestBodyJSON(r, payload)
}
