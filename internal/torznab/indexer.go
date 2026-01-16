package torznab

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Indexer interface {
	Info() znab.Info
	Search(query Query) ([]FeedItem, error)
	Download(urlStr string) (io.ReadCloser, http.Header, error)
	Capabilities() znab.Caps
}

type stremThruIndexer struct {
	info znab.Info
	caps znab.Caps
}

func (sti stremThruIndexer) Info() znab.Info {
	return sti.info
}

var lastMappedIMDBIdCached struct {
	imdbId  string
	staleAt time.Time
}

func (sti stremThruIndexer) Search(q Query) ([]FeedItem, error) {
	imdbIds := []string{}

	if q.IMDBId == "" && q.Q == "" {
		if lastMappedIMDBIdCached.staleAt.Before(time.Now()) {
			imdbId, err := imdb_torrent.GetLastMappedIMDBId()
			if err != nil {
				return nil, err
			}
			lastMappedIMDBIdCached.imdbId = imdbId
			lastMappedIMDBIdCached.staleAt = time.Now().Add(30 * time.Minute)
		}
		if lastMappedIMDBIdCached.imdbId != "" {
			imdbIds = append(imdbIds, lastMappedIMDBIdCached.imdbId)
		}
	} else if q.IMDBId == "" && q.Q != "" {
		category := imdb_title.SearchTitleTypeUnknown
		hasMovieCat, hasTvCat := q.HasMovies(), q.HasTVShows()
		if hasMovieCat && !hasTvCat {
			category = imdb_title.SearchTitleTypeMovie
		} else if !hasMovieCat && hasTvCat {
			category = imdb_title.SearchTitleTypeShow
		}
		ids, err := imdb_title.SearchIds(q.Q, category, q.Year, false, 5)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			log.Debug("no imdb ids found for query", "q", q.Q)
		}
		imdbIds = append(imdbIds, ids...)
	} else {
		imdbIds = append(imdbIds, q.IMDBId)
	}

	if len(imdbIds) == 0 {
		return []FeedItem{}, nil
	}

	var wg sync.WaitGroup
	for _, imdbId := range imdbIds {
		wg.Go(func() {
			worker_queue.TorznabIndexerSyncerQueue.Queue(worker_queue.TorznabIndexerSyncerQueueItem{
				SId: imdbId,
			})
			buddy.PullTorrentsByStremId(imdbId, "")
		})
	}
	wg.Wait()

	args := []any{}
	var query strings.Builder
	query.WriteString(
		fmt.Sprintf(
			"SELECT ito.%s, %s FROM %s ito INNER JOIN %s ti ON ti.%s = ito.%s WHERE ito.%s ",
			imdb_torrent.Column.TId,
			db.JoinPrefixedColumnNames("ti.", torrent_info.Columns...),
			imdb_torrent.TableName,
			torrent_info.TableName,
			torrent_info.Column.Hash,
			imdb_torrent.Column.Hash,
			imdb_torrent.Column.TId,
		),
	)
	if len(imdbIds) == 1 {
		query.WriteString("= ?")
	} else {
		query.WriteString("IN (" + util.RepeatJoin("?", len(imdbIds), ",") + ")")
	}
	for _, imdbId := range imdbIds {
		args = append(args, imdbId)
	}
	if q.Season != "" {
		query.WriteString(
			fmt.Sprintf(
				" AND (ti.%s = ? OR CONCAT(',', ti.%s, ',') LIKE ?)",
				torrent_info.Column.Seasons,
				torrent_info.Column.Seasons,
			),
		)
		args = append(args, q.Season, "%,"+q.Season+",%")
	}
	if q.Ep != "" {
		if q.Season != "" {
			query.WriteString(
				fmt.Sprintf(
					" AND (ti.%s = '' OR ti.%s = ? OR CONCAT(',', ti.%s, ',') LIKE ?)",
					torrent_info.Column.Episodes,
					torrent_info.Column.Episodes,
					torrent_info.Column.Episodes,
				),
			)
			args = append(args, q.Ep, "%,"+q.Ep+",%")
		} else {
			query.WriteString(
				fmt.Sprintf(
					" AND (ti.%s = ? OR CONCAT(',', ti.%s, ',') LIKE ?)",
					torrent_info.Column.Episodes,
					torrent_info.Column.Episodes,
				),
			)
			args = append(args, q.Ep, "%,"+q.Ep+",%")
		}
	}
	query.WriteString(
		fmt.Sprintf(
			" AND ti.%s = %s AND ti.%s != -1",
			torrent_info.Column.Private, db.BooleanFalse,
			torrent_info.Column.Size,
		),
	)
	rows, err := db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []FeedItem{}
	for rows.Next() {
		var imdbId string
		var tInfo torrent_info.TorrentInfo
		if err := rows.Scan(
			&imdbId,

			&tInfo.Hash,
			&tInfo.TorrentTitle,

			&tInfo.Indexer,
			&tInfo.Source,
			&tInfo.Category,
			&tInfo.CreatedAt,
			&tInfo.UpdatedAt,
			&tInfo.ParsedAt,
			&tInfo.ParserVersion,
			&tInfo.ParserInput,

			&tInfo.Seeders,
			&tInfo.Leechers,
			&tInfo.Private,

			&tInfo.Audio,
			&tInfo.BitDepth,
			&tInfo.Channels,
			&tInfo.Codec,
			&tInfo.Commentary,
			&tInfo.Complete,
			&tInfo.Container,
			&tInfo.Convert,
			&tInfo.Date,
			&tInfo.Documentary,
			&tInfo.Dubbed,
			&tInfo.Edition,
			&tInfo.EpisodeCode,
			&tInfo.Episodes,
			&tInfo.Extended,
			&tInfo.Extension,
			&tInfo.Group,
			&tInfo.HDR,
			&tInfo.Hardcoded,
			&tInfo.Languages,
			&tInfo.Network,
			&tInfo.Proper,
			&tInfo.Quality,
			&tInfo.Region,
			&tInfo.ReleaseTypes,
			&tInfo.Remastered,
			&tInfo.Repack,
			&tInfo.Resolution,
			&tInfo.Retail,
			&tInfo.Seasons,
			&tInfo.Site,
			&tInfo.Size,
			&tInfo.Subbed,
			&tInfo.ThreeD,
			&tInfo.Title,
			&tInfo.Uncensored,
			&tInfo.Unrated,
			&tInfo.Upscaled,
			&tInfo.Volumes,
			&tInfo.Year,
			&tInfo.YearEnd,
		); err != nil {
			return nil, err
		}
		var category Category
		switch tInfo.Category {
		case torrent_info.TorrentInfoCategoryMovie:
			category = CategoryMovies
		case torrent_info.TorrentInfoCategorySeries:
			category = CategoryTV
		case torrent_info.TorrentInfoCategoryXXX:
			category = CategoryXXX
		default:
			category = CategoryOther
		}
		audio := strings.Join(tInfo.Audio, ", ")
		if len(tInfo.Channels) > 0 {
			audio += " | " + strings.Join(tInfo.Channels, ", ")
		}
		items = append(items, FeedItem{
			Audio:       audio,
			Category:    category,
			Codec:       tInfo.Codec,
			IMDB:        imdbId,
			InfoHash:    tInfo.Hash,
			Language:    strings.Join(tInfo.Languages, ", "),
			Leechers:    tInfo.Leechers,
			PublishDate: tInfo.CreatedAt.Time,
			Resolution:  tInfo.Resolution,
			Seeders:     tInfo.Seeders,
			Site:        tInfo.Site,
			Size:        tInfo.Size,
			Title:       tInfo.TorrentTitle,
			Year:        tInfo.Year,
		})
	}

	if q.Offset > 0 {
		items = items[min(q.Offset, len(items)):]
	}

	if q.Limit > 0 {
		items = items[:min(q.Limit, len(items))]
	}

	return items, nil
}

func (sti stremThruIndexer) Download(urlStr string) (io.ReadCloser, http.Header, error) {
	return nil, nil, nil
}

func (sti stremThruIndexer) Capabilities() znab.Caps {
	return sti.caps
}

var StremThruIndexer = stremThruIndexer{
	info: znab.Info{
		Title:       "StremThru",
		Description: "StremThru Torznab",
	},
	caps: znab.Caps{
		Server: &znab.CapsServer{
			Title:     "StremThru",
			Strapline: "StremThru Torznab",
			Image:     "https://emojiapi.dev/api/v1/sparkles/256.png",
			URL:       config.BaseURL.String(),
			Version:   "1.3",
		},
		Searching: &znab.CapsSearching{
			Search: &znab.CapsSearchingItem{
				Available:       true,
				SupportedParams: []string{"q"},
			},
			TVSearch: &znab.CapsSearchingItem{
				Available:       true,
				SupportedParams: []string{"q,imdbid,season,ep"},
			},
			MovieSearch: &znab.CapsSearchingItem{
				Available:       true,
				SupportedParams: []string{"q,imdbid"},
			},
		},
		Categories: []znab.CapsCategory{
			{
				Category: CategoryMovies,
			},
			{
				Category: CategoryTV,
			},
		},
	},
}
