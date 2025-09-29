package animetosho

import (
	"path"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var STORAGE_BASE_URL = util.MustDecodeBase64("aHR0cHM6Ly9zdG9yYWdlLmFuaW1ldG9zaG8ub3Jn")

func GetFileLinksURL(date time.Time) string {
	return STORAGE_BASE_URL + "/dbexport/filelinks-" + date.Format(strings.ReplaceAll(time.DateOnly, "-", "")) + ".txt.xz"
}

type AnimeToshoTorrent struct {
	Magnet       string // magnet link of torrent, either obtained from source or generated from torrent file
	AId          string // related AniDB anime ID
	TorrentName  string // Name extracted from torrent file
	TorrentFiles int    // Number of files found in the torrent
	TotalSize    int64  // total size of all files in torrent, in bytes
}

func SyncDateset() {
	log := logger.Scoped("animetosho/dataset")

	writer := util.NewDatasetWriter(util.DatasetWriterConfig[any]{
		BatchSize: 500,
		Log:       log,
		Upsert: func(titles []any) error {
			return nil
		},
		SleepDuration: 200 * time.Millisecond,
	})

	ds := util.NewTSVDataset(&util.TSVDatasetConfig[any]{
		DatasetConfig: util.DatasetConfig{
			Archive:     "gz",
			DownloadDir: path.Join(config.DataDir, "imdb"),
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-24 * time.Hour))
			},
			Log: log,
			URL: STORAGE_BASE_URL + "/dbexport/torrents-latest.txt.xz",
		},
		GetRowKey: func(row []string) string {
			return row[0]
		},
		HasHeaders: true,
		IsValidHeaders: func(headers []string) bool {
			return slices.Equal(headers, []string{
				"id",
				"tosho_id",
				"nyaa_id",
				"anidex_id",
				"name",
				"link",
				"magnet",
				"cat",
				"website",
				"totalsize",
				"date_posted",
				"comment",
				"date_added",
				"date_completed",
				"torrentname",
				"torrentfiles",
				"stored_nzb",
				"stored_torrent",
				"nyaa_class",
				"nyaa_cat",
				"anidex_cat",
				"anidex_labels",
				"btih",
				"btih_sha256",
				"isdupe",
				"deleted",
				"date_updated",
				"aid",
				"eid",
				"fid",
				"gids",
				"resolveapproved",
				"main_fileid",
				"srcurl",
				"srcurltype",
				"srctitle",
				"status",
			})
		},
		ParseRow: func(row []string) (*IMDBTitle, error) {
			nilValue := ``

			tType, err := util.TSVGetValue(row, 1, "", nilValue)
			if err != nil {
				return nil, err
			}

			return &IMDBTitle{
				TId:       tId,
				Title:     title,
				OrigTitle: origTitle,
				Year:      year,
				Type:      tType,
				IsAdult:   isAdult,
			}, nil
		},
		Writer: writer,
	})
}
