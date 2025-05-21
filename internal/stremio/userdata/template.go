package stremio_userdata

import "github.com/MunifTanjim/stremthru/internal/stremio/configure"

type TemplateDataUserData struct {
	SavedUserDataKey     string
	SavedUserDataOptions []configure.ConfigOption
	IsRedacted           bool
}
