package torznab_client

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
	"github.com/anacrolix/torrent/metainfo"
)

type torzFileCached struct {
	Hash       string
	MagnetLink string
	Private    bool
	Files      []torrent_stream.File
}

var torzFileCache = cache.NewCache[torzFileCached](&cache.CacheConfig{
	Lifetime: 6 * time.Hour,
	Name:     "torznab:indexer:file",
	MaxSize:  5120,
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

	// TODO: use shared.FetchTorrentFile
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

func TorzFromChannelItem(o *znab.ChannelItem, attrs znab.ChannelItemAttrs) *Torz {
	t := &Torz{}
	t.Hash = strings.ToLower(attrs.Get(znab.TorznabAttrNameInfoHash))
	t.Title = o.Title
	if size, err := strconv.ParseInt(attrs.Get(znab.TorznabAttrNameSize), 10, 64); err == nil && size > 0 {
		t.Size = size
	} else if o.Size > 0 {
		t.Size = o.Size
	} else if o.Enclosure.Length > 0 {
		t.Size = o.Enclosure.Length
	}
	t.Seeders = util.SafeParseInt(attrs.Get(znab.TorznabAttrNameSeeders), 0)
	if leechers := util.SafeParseInt(attrs.Get(znab.TorznabAttrNameLeechers), 0); leechers > 0 {
		t.Leechers = leechers
	} else if peers := util.SafeParseInt(attrs.Get(znab.TorznabAttrNamePeers), 0); peers > t.Seeders {
		t.Leechers = peers - t.Seeders
	}
	if minr := util.SafeParseFloat(attrs.Get(znab.TorznabAttrNameMinimumRatio), 0); minr > 0 {
		t.Private = true
	} else if minst := util.SafeParseFloat(attrs.Get(znab.TorznabAttrNameMinimumSeedTime), 0); minst > 0 {
		t.Private = true
	}
	if strings.HasPrefix(o.Enclosure.URL, "magnet:?") {
		t.MagnetLink = o.Enclosure.URL
		if t.Hash == "" {
			if m, err := core.ParseMagnetLink(t.MagnetLink); err == nil {
				t.Hash = m.Hash
			}
		}
	} else if magnetUrl := attrs.Get(znab.TorznabAttrNameMagnetURL); strings.HasPrefix(magnetUrl, "magnet:?") {
		t.MagnetLink = magnetUrl
		if t.Hash == "" {
			if m, err := core.ParseMagnetLink(t.MagnetLink); err == nil {
				t.Hash = m.Hash
			}
		}
	} else if strings.HasPrefix(o.Enclosure.URL, "http") {
		t.SourceLink = o.Enclosure.URL
	}
	return t
}

type Indexer interface {
	GetId() string
	GetName() string
	NewSearchQuery(fn func(caps Caps) Function) (*Query, error)
	Search(query url.Values) ([]Torz, error)
}
