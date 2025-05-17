package stremio_wrap

import (
	"github.com/MunifTanjim/stremthru/internal/logger"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

var log = logger.Scoped("stremio/wrap")

var LogError = stremio_shared.LogError
