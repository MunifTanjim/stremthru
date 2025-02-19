package stremio_wrap

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/MunifTanjim/stremthru/store"
)

type resolvedStore struct {
	store     store.Store
	authToken string
}

type multiStore []resolvedStore

type multiStoreResultItem[T any] struct {
	Data T
	Err  error
}

type multiStoreResult[T any] struct {
	Data   []T
	Err    []error
	HasErr bool
}

func (ms multiStore) GetUser() multiStoreResult[*store.User] {
	count := len(ms)
	res := multiStoreResult[*store.User]{
		Data:   make([]*store.User, count),
		Err:    make([]error, count),
		HasErr: false,
	}

	var wg sync.WaitGroup
	for i := range ms {
		wg.Add(1)
		s := &ms[i]
		go func() {
			defer wg.Done()
			params := &store.GetUserParams{}
			params.APIKey = s.authToken
			res.Data[i], res.Err[i] = s.store.GetUser(params)
			if res.Err[i] != nil {
				res.HasErr = true
			}
		}()
	}
	wg.Wait()

	return res
}

type multiStoreCheckMagnetData struct {
	ByHash map[string]string
	Err    []error
	HasErr bool
	m      sync.Mutex
}

func (ms multiStore) CheckMagnet(params *store.CheckMagnetParams, log *slog.Logger) *multiStoreCheckMagnetData {
	storeCount := len(ms)
	res := multiStoreCheckMagnetData{
		ByHash: map[string]string{},
		Err:    make([]error, storeCount),
		HasErr: false,
	}

	firstStore := ms[0]

	missingHashes := []string{}

	cmParams := &store.CheckMagnetParams{
		Magnets:  params.Magnets,
		ClientIP: params.ClientIP,
		SId:      params.SId,
	}
	cmParams.APIKey = firstStore.authToken
	if cmRes, err := firstStore.store.CheckMagnet(cmParams); err != nil {
		log.Error("Failed to check magnet", "store", firstStore.store.GetName(), "error", err)
		res.Err[0] = err
		res.HasErr = true

		if storeCount > 1 {
			missingHashes = params.Magnets
		}
	} else {
		storeCode := strings.ToUpper(string(firstStore.store.GetName().Code()))
		for i := range cmRes.Items {
			item := cmRes.Items[i]
			if item.Status == store.MagnetStatusCached {
				res.ByHash[item.Hash] = storeCode
			} else if storeCount > 1 {
				missingHashes = append(missingHashes, item.Hash)
			}
		}
	}

	if storeCount == 1 || len(missingHashes) == 0 {
		return &res
	}

	var wg sync.WaitGroup
	for i := range storeCount - 1 {
		idx := i + 1
		s := &ms[idx]

		wg.Add(1)
		go func() {
			defer wg.Done()
			cmParams := &store.CheckMagnetParams{
				Magnets:  missingHashes,
				ClientIP: params.ClientIP,
				SId:      params.SId,
			}
			cmParams.APIKey = s.authToken
			cmRes, err := s.store.CheckMagnet(cmParams)
			if err != nil {
				log.Warn("Failed to check magnet", "store", s.store.GetName(), "error", err)
				res.Err[idx] = err
				res.HasErr = true
			} else {
				res.m.Lock()
				defer res.m.Unlock()

				storeCode := strings.ToUpper(string(s.store.GetName().Code()))
				for _, item := range cmRes.Items {
					if _, found := res.ByHash[item.Hash]; !found && item.Status == store.MagnetStatusCached {
						res.ByHash[item.Hash] = storeCode
					}
				}
			}
		}()
	}
	wg.Wait()

	return &res
}
