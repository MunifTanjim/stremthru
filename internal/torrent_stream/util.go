package torrent_stream

import (
	"path/filepath"

	"github.com/anacrolix/torrent/metainfo"
)

func FilesFromTorrentInfo(info *metainfo.Info) (files Files) {
	for i, f := range info.Files {
		path := "/" + f.DisplayPath(info)
		files = append(files, File{
			Path:   path,
			Idx:    i,
			Size:   f.Length,
			Name:   filepath.Base(path),
			Source: "tor",
		})
	}
	return files
}
