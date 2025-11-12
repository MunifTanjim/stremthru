package torznab_client

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/anacrolix/torrent/metainfo"
)

type torzFileCached struct {
	Hash       string
	MagnetLink string
	Private    bool
	Files      []torrent_stream.File
}

var torzFileCache = cache.NewCache[torzFileCached](&cache.CacheConfig{
	Lifetime:      6 * time.Hour,
	Name:          "torznab:indexer:file",
	LocalCapacity: 5120,
})

type Torz struct {
	Indexer string

	Hash  string
	Title string
	Size  int64

	Seeders  int
	Leechers int
	Private  bool

	Files []torrent_stream.File

	MagnetLink string
	SourceLink string
}

func (t *Torz) HasMissingData() bool {
	return t.Hash == "" || t.MagnetLink == ""
}

func (t *Torz) EnsureMagnet() error {
	if !t.HasMissingData() {
		return nil
	}

	if t.SourceLink == "" {
		return errors.New("no source link to generate magnet")
	}

	cachedTorz := torzFileCached{}
	if torzFileCache.Get(t.SourceLink, &cachedTorz) {
		t.Hash = cachedTorz.Hash
		t.MagnetLink = cachedTorz.MagnetLink
		if cachedTorz.Private {
			t.Private = true
		}
		t.Files = cachedTorz.Files
		return nil
	}

	client := config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Get(t.SourceLink)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if http.StatusMovedPermanently <= resp.StatusCode && resp.StatusCode <= http.StatusSeeOther {
		if location := resp.Header.Get("Location"); strings.HasPrefix(location, "magnet:?") {
			m, err := core.ParseMagnetLink(location)
			if err != nil {
				return err
			}
			t.Hash = m.Hash
			t.MagnetLink = m.RawLink

			cachedTorz.Hash = t.Hash
			cachedTorz.MagnetLink = t.MagnetLink
			torzFileCache.Add(t.SourceLink, cachedTorz)
			return nil
		}
	}

	mi, err := metainfo.Load(resp.Body)
	if err != nil {
		return err
	}

	m, err := mi.MagnetV2()
	if err != nil {
		return err
	}
	if !m.InfoHash.Ok {
		return errors.New("unsupported torrent file: only v1 torrents are supported")
	}

	t.Hash = strings.ToLower(m.InfoHash.Value.String())
	t.MagnetLink = m.String()

	mii, err := mi.UnmarshalInfo()
	if err != nil {
		return err
	}

	t.Files = torrent_stream.FilesFromTorrentInfo(&mii)

	if mii.Private != nil && *mii.Private {
		t.Private = true
	}

	cachedTorz.Hash = t.Hash
	cachedTorz.MagnetLink = t.MagnetLink
	cachedTorz.Private = t.Private
	cachedTorz.Files = t.Files
	torzFileCache.Add(t.SourceLink, cachedTorz)

	return nil
}

type Indexer interface {
	GetId() string
	NewSearchQuery(fn func(caps Caps) Function) (*Query, error)
	Search(query *Query) ([]Torz, error)
}
