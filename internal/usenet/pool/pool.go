package usenet_pool

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

type ProviderConfig struct {
	nntp.PoolConfig
	IsBackup bool
}

type Config struct {
	Log                  *logger.Logger
	Providers            []ProviderConfig
	RequiredCapabilities []string
	MinConnections       int
	CacheSize            int64
}

func (conf *Config) setDefaults() {
	if conf.Log == nil {
		conf.Log = logger.Scoped("usenet/pool")
	}
	slices.SortStableFunc(conf.Providers, func(a, b ProviderConfig) int {
		if a.IsBackup && !b.IsBackup {
			return 1
		}
		if !a.IsBackup && b.IsBackup {
			return -1
		}
		return 0
	})
	if conf.CacheSize == 0 {
		conf.CacheSize = 200 * 1024 * 1024 // 200 MB
	}
}

type providerPool struct {
	*nntp.Pool
	isBackup bool
}

type Pool struct {
	Log                  *logger.Logger
	providers            []*providerPool
	providersMutex       sync.RWMutex
	requiredCapabilities []string
	minConnections       int
	fetchGroup           singleflight.Group
	segmentCache         *SegmentCache
}

func NewPool(conf *Config) (*Pool, error) {
	conf.setDefaults()

	pools := []*providerPool{}

	for i := range conf.Providers {
		provider := &conf.Providers[i]
		if provider.Log == nil {
			provider.Log = conf.Log.With("id", provider.Id())
		}
		pool, err := nntp.NewPool(&provider.PoolConfig)
		if err != nil {
			return nil, err
		}
		pools = append(pools, &providerPool{
			Pool:     pool,
			isBackup: provider.IsBackup,
		})
	}

	up := &Pool{
		Log:                  conf.Log,
		providers:            pools,
		requiredCapabilities: conf.RequiredCapabilities,
		minConnections:       conf.MinConnections,
		segmentCache:         NewSegmentCache(conf.CacheSize),
	}

	up.verifyProviders()

	if err := up.ensureMinSize(context.Background()); err != nil {
		up.Log.Warn("failed to ensure min size at startup", "error", err)
	}

	return up, nil
}

func (p *Pool) ensureMinSize(ctx context.Context) error {
	if p.minConnections == 0 {
		return nil
	}

	currentCount := 0
	for _, provider := range p.providers {
		if provider.IsOnline() {
			currentCount += int(provider.Stat().TotalResources())
		}
	}

	if currentCount >= p.minConnections {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for range p.minConnections - currentCount {
		c, err := p.GetConnection(ctx, nil, false)
		if err != nil {
			return err
		}
		c.Release()
	}
	return nil
}

func (p *Pool) verifyProviders() {
	if len(p.requiredCapabilities) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, provider := range p.providers {
		wg.Go(func() {
			c, err := provider.Acquire(context.Background())
			if err != nil {
				return
			}
			defer c.Release()

			caps, err := c.Capabilities()
			if err != nil {
				provider.SetState(nntp.PoolStateOffline)
				return
			}

			for _, capability := range p.requiredCapabilities {
				if !slices.Contains(caps.Capabilities, capability) {
					provider.SetState(nntp.PoolStateDisabled)
					p.Log.Warn("disabling provider pool due to missing required capability", "capability", capability, "id", provider.Id())
					return
				}
			}
		})
	}
	wg.Wait()
}

func (p *Pool) GetConnection(ctx context.Context, excludeProvider []string, includeBackup bool) (*nntp.PooledConnection, error) {
	p.providersMutex.RLock()
	if len(p.providers) == 0 {
		p.providersMutex.RUnlock()
		return nil, errors.New("usenet: no providers configured")
	}
	providers := make([]*providerPool, 0, len(p.providers))
	for _, provider := range p.providers {
		if !provider.IsOnline() {
			continue
		}
		if provider.isBackup && !includeBackup {
			continue
		}
		if slices.Contains(excludeProvider, provider.Id()) {
			continue
		}
		providers = append(providers, provider)
	}
	p.providersMutex.RUnlock()

	if len(providers) == 0 {
		return nil, errors.New("usenet: no available providers")
	}

	for _, provider := range providers {
		if provider.Stat().AcquiredResources() == provider.MaxSize() {
			continue
		}
		return provider.Acquire(ctx)
	}

	return providers[0].Acquire(ctx)
}

func isArticleNotFoundError(err error) bool {
	var nntpErr *nntp.Error
	if errors.As(err, &nntpErr) {
		return nntpErr.Code == nntp.ErrorCodeNoSuchArticle
	}
	return false
}

func (p *Pool) ensureConnectionGroup(conn *nntp.PooledConnection, groups ...string) error {
	if len(groups) == 0 {
		return nil
	}
	errs := []error{}
	currGroup := conn.CurrentGroup()
	for _, group := range groups {
		if group == currGroup {
			return nil
		}
		_, err := conn.Group(group)
		if err == nil {
			p.Log.Trace("switched connection current group", "group", group)
			return nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (p *Pool) fetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (SegmentData, error) {
	messageId := segment.MessageId
	if cachedData, ok := p.segmentCache.Get(messageId); ok {
		p.Log.Trace("fetch segment - cache hit", "segment_num", segment.Number, "message_id", messageId, "size", len(cachedData.Body()))
		return cachedData, nil
	}

	result, err, _ := p.fetchGroup.Do(messageId, func() (any, error) {
		var excludeProviders []string
		errs := []error{}
		failedAttempts := 0

		for failedAttempts < 3 {
			if len(excludeProviders) > 0 {
				p.Log.Trace("fetch segment - retry", "segment_num", segment.Number, "message_id", messageId, "failed_attempts", failedAttempts, "excluded_providers", len(excludeProviders))
			}

			conn, err := p.GetConnection(ctx, excludeProviders, len(excludeProviders) > 0)
			if err != nil {
				errs = append(errs, err)
				failedAttempts++
				p.Log.Warn("fetch segment - failed to get connection", "error", err, "segment_num", segment.Number, "message_id", messageId)
				continue
			}

			p.Log.Trace("fetch segment - connection acquired", "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId())

			if err := p.ensureConnectionGroup(conn, groups...); err != nil {
				conn.Release()
				errs = append(errs, err)
				failedAttempts++
				p.Log.Warn("fetch segment - failed to ensure group", "error", err, "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId())
				continue
			}

			article, err := conn.Body("<" + messageId + ">")
			if err != nil {
				errs = append(errs, err)
				if isArticleNotFoundError(err) {
					conn.Release()
					excludeProviders = append(excludeProviders, conn.ProviderId())
					p.Log.Trace("fetch segment - article not found", "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId())
					continue
				}

				conn.Destroy()
				failedAttempts++
				p.Log.Warn("fetch segment - failed to get body", "error", err, "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId())
				continue
			}

			p.Log.Trace("fetch segment - got body", "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId())

			decoder := NewYEncDecoder(article.Body)
			defer decoder.Close()

			data, err := decoder.ReadAll()

			conn.Release()

			if err != nil {
				errs = append(errs, err)
				failedAttempts++
				p.Log.Warn("fetch segment - failed to decode", "error", err, "segment_num", segment.Number, "message_id", messageId)
				continue
			}

			p.Log.Trace("fetch segment - decoded body", "segment_num", segment.Number, "message_id", messageId, "decoded_size", len(data.body))

			p.segmentCache.Set(messageId, data)

			return data, nil
		}

		return nil, errors.New("failed to fetch segment " + strconv.Itoa(segment.Number) + " <" + messageId + "> after retries: " + errors.Join(errs...).Error())
	})

	if err != nil {
		return nil, err
	}

	return result.(SegmentData), nil
}

func (p *Pool) Close() {
	p.providersMutex.Lock()
	defer p.providersMutex.Unlock()

	for _, provider := range p.providers {
		provider.Close()
	}
}
