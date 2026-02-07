package stremio_shared

import (
	"github.com/MunifTanjim/stremthru/internal/logger"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
)

type Ctx struct {
	storecontext.Context
	Log *logger.Logger
}
