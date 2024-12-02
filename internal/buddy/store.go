package buddy

import (
	"log"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/store"
)

var Client = NewAPIClient(&APIClientConfig{
	BaseURL: config.BuddyURL,
	APIKey:  config.BuddyAuthToken,
})

var checkMagnetCache = func() cache.Cache[store.CheckMagnetDataItem] {
	return cache.NewCache[store.CheckMagnetDataItem](&cache.CacheConfig{
		Name:     "buddy:checkMagnet",
		Lifetime: 10 * time.Minute,
	})
}()

func getCachedCheckMagnet(s store.Store, magnetHash string) *store.CheckMagnetDataItem {
	v := &store.CheckMagnetDataItem{}
	if ok := checkMagnetCache.Get(string(s.GetName())+":"+magnetHash, v); ok {
		return v
	}
	return nil
}

func setCachedCheckMagnet(s store.Store, magnetHash string, v *store.CheckMagnetDataItem) {
	checkMagnetCache.Add(string(s.GetName())+":"+magnetHash, *v)
}

func TrackMagnet(s store.Store, hash string, files []store.MagnetFile, cacheMiss bool) {
	if !Client.IsAvailable() {
		return
	}

	if _, err := Client.TrackMagnetCache(&TrackMagnetCacheParams{
		Store:     s.GetName(),
		Hash:      hash,
		Files:     files,
		CacheMiss: cacheMiss,
	}); err != nil {
		log.Printf("failed to track magnet cache for %s:%s: %v\n", s.GetName(), hash, err)
	}
}

func CheckMagnet(s store.Store, hashes []string) (*store.CheckMagnetData, error) {
	uncachedHashes := []string{}

	data := &store.CheckMagnetData{
		Items: []store.CheckMagnetDataItem{},
	}

	for _, hash := range hashes {
		if v := getCachedCheckMagnet(s, hash); v != nil {
			data.Items = append(data.Items, *v)
		} else {
			uncachedHashes = append(uncachedHashes, hash)
		}
	}

	if Client.IsAvailable() && len(uncachedHashes) > 0 {
		res, err := Client.CheckMagnetCache(&CheckMagnetCacheParams{
			Store:  s.GetName(),
			Hashes: uncachedHashes,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range res.Data.Items {
			setCachedCheckMagnet(s, item.Hash, &item)
			data.Items = append(data.Items, item)
		}
	}

	return data, nil
}
