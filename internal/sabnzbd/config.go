package sabnzbd

import "github.com/MunifTanjim/stremthru/internal/config"

var version = config.Newz.GetSABnzbdVersion()

var categories = []map[string]any{
	{
		"name":     "*",
		"order":    0,
		"pp":       "3",
		"script":   "None",
		"dir":      "",
		"newzbin":  "",
		"priority": 0,
	},
	{
		"name":     "movies",
		"order":    1,
		"pp":       "",
		"script":   "Default",
		"dir":      "",
		"newzbin":  "",
		"priority": -100,
	},
	{
		"name":     "tv",
		"order":    2,
		"pp":       "",
		"script":   "Default",
		"dir":      "",
		"newzbin":  "",
		"priority": -100,
	},
	{
		"name":     "audio",
		"order":    3,
		"pp":       "",
		"script":   "Default",
		"dir":      "",
		"newzbin":  "",
		"priority": -100,
	},
	{
		"name":     "software",
		"order":    4,
		"pp":       "",
		"script":   "Default",
		"dir":      "",
		"newzbin":  "",
		"priority": -100,
	},
}

var servers = []map[string]any{}
