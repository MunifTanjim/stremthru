package dash_api

import (
	"github.com/MunifTanjim/stremthru/internal/server"
)

type Error = server.Error

var ErrorBadRequest = server.ErrorBadRequest
var ErrorForbidden = server.ErrorForbidden
var ErrorInternalServerError = server.ErrorInternalServerError
var ErrorLocked = server.ErrorLocked
var ErrorMethodNotAllowed = server.ErrorMethodNotAllowed
var ErrorNotFound = server.ErrorNotFound
var ErrorUnauthorized = server.ErrorUnauthorized
var ErrorUnsupportedMediaType = server.ErrorUnsupportedMediaType
