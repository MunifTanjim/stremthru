package animetosho

import (
	"database/sql"
	"errors"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var datasetSyncMutex sync.Mutex

var datasetDownloadDir = path.Join(config.DataDir, "animetosho")

var ErrDatasetSyncInProgress = errors.New("dataset sync in progress")

func PurgeDatasetTemporaryFiles() error {
	if !datasetSyncMutex.TryLock() {
		return ErrDatasetSyncInProgress
	}
	defer datasetSyncMutex.Unlock()

	return os.RemoveAll(datasetDownloadDir)
}

func GetDatasetTemporaryDir() string {
	return datasetDownloadDir
}

var STORAGE_BASE_URL = util.MustDecodeBase64("aHR0cHM6Ly9zdG9yYWdlLmFuaW1ldG9zaG8ub3Jn")

func GetFileLinksURL(date time.Time) string {
	return STORAGE_BASE_URL + "/dbexport/filelinks-" + date.Format(strings.ReplaceAll(time.DateOnly, "-", "")) + ".txt.xz"
}

type AnimeToshoTorrent struct {
	Id           int64  // unique identifier
	Magnet       string // magnet link of torrent, either obtained from source or generated from torrent file
	TotalSize    int64  // total size of all files in torrent, in bytes
	TorrentName  string // Name extracted from torrent file
	TorrentFiles int    // Number of files found in the torrent
	AId          string // related AniDB anime ID

	Files []AnimeToshoTorrentFile
}

type AnimeToshoTorrentFile struct {
	Id        int64  // unique identifier
	TorrentId int64  // ID of associated torrent entry
	Filename  string // file's name; includes path if supplied
	FileSize  int64  // file's size in bytes
}

func SyncDataset() error {
	log := logger.Scoped("animetosho/dataset")

	if !datasetSyncMutex.TryLock() {
		log.Warn("dataset sync already in progress, skipping")
		return nil
	}
	defer datasetSyncMutex.Unlock()

	flog := logger.Scoped("animetosho/dataset/files")

	filesDB, err := sql.Open("sqlite3", "file:animetosho-files.db?mode=memory")
	if err != nil {
		return err
	}
	defer filesDB.Close()

	if _, err := filesDB.Exec(`
		BEGIN;
		CREATE TABLE files (
			id int NOT NULL,
			torrent_id int NOT NULL,
			filename varchar NOT NULL,
			filesize int NOT NULL,
			PRIMARY KEY (id)
		);
		CREATE INDEX files_torrent_id ON files (torrent_id);
		COMMIT;
	`); err != nil {
		return err
	}

	filesDS := util.NewSimpleTSVDataset(&util.SimpleTSVDatasetConfig[AnimeToshoTorrentFile]{
		DatasetConfig: util.DatasetConfig{
			Archive:     "xz",
			DownloadDir: path.Join(datasetDownloadDir, "files"),
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-24 * time.Hour))
			},
			Log: flog,
			URL: STORAGE_BASE_URL + "/dbexport/files-latest.txt.xz",
		},
		GetRowKey: func(row []string) string {
			return row[0]
		},
		HasHeaders: true,
		IsValidHeaders: func(headers []string) bool {
			return slices.Equal(headers, []string{
				"id",
				"torrent_id",
				"is_archive",
				"filename",
				"filesize",
				"vidframes",
				"crc32",
				"md5",
				"sha1",
				"sha256",
				"tth",
				"ed2k",
				"bt2",
				"crc32k",
				"torpc_sha1_16k",
				"torpc_sha1_32k",
				"torpc_sha1_64k",
				"torpc_sha1_128k",
				"torpc_sha1_256k",
				"torpc_sha1_512k",
				"torpc_sha1_1024k",
				"torpc_sha1_2048k",
				"torpc_sha1_4096k",
				"torpc_sha1_8192k",
				"torpc_sha1_16384k",
			})
		},
		NoDiff: true,
		ParseRow: func(row []string) (*AnimeToshoTorrentFile, error) {
			nilValue := ``

			isArchived, err := util.TSVGetValue(row, 2, 0, nilValue)
			if err != nil {
				return nil, err
			}

			if isArchived == 1 {
				return nil, nil
			}

			id, err := util.TSVGetValue(row, 0, int64(0), nilValue)
			if err != nil {
				return nil, err
			}

			torrentId, err := util.TSVGetValue(row, 1, int64(0), nilValue)
			if err != nil {
				return nil, err
			}

			fileName, err := util.TSVGetValue(row, 3, "", nilValue)
			if err != nil {
				return nil, err
			}

			fileSize, err := util.TSVGetValue(row, 4, int64(0), nilValue)
			if err != nil {
				return nil, err
			}

			return &AnimeToshoTorrentFile{
				Id:        id,
				TorrentId: torrentId,
				Filename:  fileName,
				FileSize:  fileSize,
			}, nil
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[AnimeToshoTorrentFile]{
			BatchSize: 1000,
			Log:       flog,
			Upsert: func(files []AnimeToshoTorrentFile) error {
				count := len(files)
				if count == 0 {
					return nil
				}

				var query strings.Builder
				query.WriteString("INSERT INTO files (id,torrent_id,filename,filesize) VALUES ")
				query.WriteString(util.RepeatJoin("(?,?,?,?)", count, ","))
				query.WriteString(";")

				args := make([]any, 0, count*4)
				for _, f := range files {
					args = append(args, f.Id, f.TorrentId, f.Filename, f.FileSize)
				}

				_, err := filesDB.Exec(query.String(), args...)
				return err
			},
			SleepDuration: 3 * time.Millisecond,
		}),
	})

	if err := filesDS.Process(); err != nil {
		return err
	}

	ds := util.NewSimpleTSVDataset(&util.SimpleTSVDatasetConfig[AnimeToshoTorrent]{
		DatasetConfig: util.DatasetConfig{
			Archive:     "xz",
			DownloadDir: path.Join(datasetDownloadDir, "torrents"),
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
				"nekobt_id",
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
				"nekobt_hide",
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
		ParseRow: func(row []string) (*AnimeToshoTorrent, error) {
			nilValue := ``

			id, err := util.TSVGetValue(row, 0, int64(0), nilValue)
			if err != nil {
				return nil, err
			}

			magnet, err := util.TSVGetValue(row, 7, ``, nilValue)
			if err != nil {
				return nil, err
			}

			totalSize, err := util.TSVGetValue(row, 10, int64(0), nilValue)
			if err != nil {
				return nil, err
			}

			torrentName, err := util.TSVGetValue(row, 15, ``, nilValue)
			if err != nil {
				return nil, err
			}

			if torrentName == "" {
				name, err := util.TSVGetValue(row, 5, ``, nilValue)
				if err != nil {
					return nil, err
				}
				torrentName = name
			}

			torrentFiles, err := util.TSVGetValue(row, 16, 0, nilValue)
			if err != nil {
				return nil, err
			}

			aid, err := util.TSVGetValue(row, 29, ``, nilValue)
			if err != nil {
				return nil, err
			}

			rows, err := filesDB.Query("SELECT id, torrent_id, filename, filesize FROM files WHERE torrent_id = ? ORDER BY id ASC", id)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var files []AnimeToshoTorrentFile
			for rows.Next() {
				var f AnimeToshoTorrentFile
				if err := rows.Scan(&f.Id, &f.TorrentId, &f.Filename, &f.FileSize); err != nil {
					return nil, err
				}
				files = append(files, f)
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}

			shouldDiscard := false
			if len(files) > 0 {
				hasVideoFile := false
				for i := range files {
					f := &files[i]
					if core.HasVideoExtension(f.Filename) {
						hasVideoFile = true
						break
					}
				}
				if !hasVideoFile {
					shouldDiscard = true
				}
			}

			if shouldDiscard {
				return nil, nil
			}

			return &AnimeToshoTorrent{
				Id:           id,
				Magnet:       magnet,
				TotalSize:    totalSize,
				TorrentName:  torrentName,
				TorrentFiles: torrentFiles,
				AId:          aid,
				Files:        files,
			}, nil
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[AnimeToshoTorrent]{
			BatchSize: 5000,
			Log:       log,
			Upsert: func(tors []AnimeToshoTorrent) error {
				tInfos := make([]torrent_info.TorrentInfoInsertData, 0, len(tors))
				for i := range tors {
					tor := &tors[i]
					m, err := core.ParseMagnetLink(tor.Magnet)
					if err != nil {
						log.Warn("failed to parse magnet link", "error", err, "magnet", tor.Magnet)
						continue
					}
					tInfo := torrent_info.TorrentInfoInsertData{
						Hash:         m.Hash,
						TorrentTitle: tor.TorrentName,
						Size:         tor.TotalSize,
						Source:       torrent_info.TorrentInfoSourceAnimeTosho,
						Files:        make(torrent_stream.Files, 0, len(tor.Files)),
					}
					for i := range tor.Files {
						f := &tor.Files[i]
						filename := strings.TrimPrefix(f.Filename, tor.TorrentName+"/")
						if !strings.HasPrefix(filename, "/") {
							filename = "/" + filename
						}
						tInfo.Files = append(tInfo.Files, torrent_stream.File{
							Path: filename,
							Idx:  -1,
							Size: f.FileSize,
							Name: path.Base(filename),
						})
					}
					tInfos = append(tInfos, tInfo)
				}
				return torrent_info.Upsert(tInfos, "", true)
			},
			SleepDuration: 200 * time.Millisecond,
		}),
	})

	if err := ds.Process(); err != nil {
		return err
	}

	return nil
}
