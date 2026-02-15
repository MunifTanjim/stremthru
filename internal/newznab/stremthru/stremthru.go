package newznab_stremthru

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/newznab"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

var (
	_ newznab_client.Indexer = (*Indexer)(nil)
)

type Indexer struct {
	user string
	pass string
}

func NewIndexer(apiKey string) *Indexer {
	ba, _ := util.ParseBasicAuth(apiKey)
	return &Indexer{
		user: ba.Username,
		pass: ba.Password,
	}
}

var getCaps = sync.OnceValue(func() *znab.Caps {
	caps := newznab.StremThruIndexer.Capabilities()
	return &caps
})

func (i *Indexer) GetId() string {
	return "stremthru"
}

func (i *Indexer) isValidAPIKey() bool {
	password := config.UserAuth.GetPassword(i.user)
	return password != "" && password == i.pass
}

func (i *Indexer) NewSearchQuery(fn func(caps *znab.Caps) newznab_client.Function) (*newznab_client.Query, error) {
	caps := getCaps()
	return newznab_client.NewQuery(caps).SetT(fn(caps)), nil
}

func (i *Indexer) Search(query url.Values, headers http.Header) ([]newznab_client.Newz, error) {
	if !i.isValidAPIKey() {
		return nil, fmt.Errorf("invalid credentials")
	}

	q, err := newznab.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	feedItems, err := newznab.StremThruIndexer.Search(q)
	if err != nil {
		return nil, err
	}

	result := make([]newznab_client.Newz, 0, len(feedItems))
	for _, item := range feedItems {
		result = append(result, convertFeedItemToNewz(item))
	}
	return result, nil
}

func convertFeedItemToNewz(fi newznab.FeedItem) newznab_client.Newz {
	return newznab_client.Newz{
		Title:        fi.Title,
		GUID:         fi.GUID,
		PublishDate:  fi.PublishDate,
		Size:         fi.Size,
		Files:        fi.Files,
		Poster:       fi.Poster,
		Group:        fi.Group,
		Grabs:        fi.Grabs,
		Comments:     fi.Comments,
		Password:     fi.Password,
		Date:         fi.UsenetDate,
		Categories:   []string{strconv.Itoa(fi.Category.ID)},
		IMDB:         fi.IMDB,
		Season:       fi.Season,
		Episode:      fi.Episode,
		DownloadLink: fi.Link,
		Indexer:      newznab_client.ChannelItemIndexer(fi.Indexer),
	}
}
