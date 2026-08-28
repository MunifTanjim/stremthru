package torznab_client

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
	"github.com/anacrolix/torrent/metainfo"
)

var log = logger.Scoped("torznab/client")

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
	GUID    string
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

	cacheKeys := []string{}
	if t.GUID != "" {
		cacheKeys = append(cacheKeys, util.MD5Hash(t.GUID))
	}
	if t.Hash != "" {
		cacheKeys = append(cacheKeys, t.Hash)
	}
	magnetLink, file, err := shared.FetchTorrentFile(t.SourceLink, &shared.FetchTorrentFileOptions{
		Log:       log,
		CacheKeys: cacheKeys,
	})
	if err != nil {
		return err
	}

	if magnetLink != "" {
		m, err := core.ParseMagnetLink(magnetLink)
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

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	mi, err := metainfo.Load(f)
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
	t.GUID = o.GUID
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
	if magnetUrl := attrs.Get(znab.TorznabAttrNameMagnetURL); strings.HasPrefix(magnetUrl, "magnet:?") {
		t.MagnetLink = magnetUrl
	}
	if strings.HasPrefix(o.Enclosure.URL, "magnet:?") {
		t.MagnetLink = o.Enclosure.URL
	} else if strings.HasPrefix(o.Enclosure.URL, "http") {
		t.SourceLink = o.Enclosure.URL
	}
	if t.Hash == "" && t.MagnetLink != "" {
		if m, err := core.ParseMagnetLink(t.MagnetLink); err == nil {
			t.Hash = m.Hash
		}
	}
	return t
}

type Indexer interface {
	GetId() string
	GetName() string
	NewSearchQuery(fn func(caps Caps) Function) (*Query, error)
	Search(ctx context.Context, query url.Values) ([]Torz, error)
}
