package usenet_pool

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var ErrNoProvidersAvailable = errors.New("usenet: no available providers")
var ErrArticleNotFound = errors.New("usenet: article not found")

type ProviderConfig struct {
	nntp.PoolConfig
	Priority int
	IsBackup bool
}

type Config struct {
	Log                  *logger.Logger
	Providers            []ProviderConfig
	RequiredCapabilities []string
	MinConnections       int
	SegmentCache         SegmentCache
}

func (conf *Config) setDefaults() {
	if conf.Log == nil {
		conf.Log = logger.Scoped("usenet/pool")
	}
	slices.SortStableFunc(conf.Providers, func(a, b ProviderConfig) int {
		return a.Priority - b.Priority
	})
	if conf.SegmentCache == nil {
		conf.SegmentCache = getNoopSegmentCache()
	}
}

type providerPool struct {
	*nntp.Pool
	priority int
	isBackup bool
}

type Pool struct {
	Log                  *logger.Logger
	providers            []*providerPool
	providersMutex       sync.RWMutex
	requiredCapabilities []string
	minConnections       int
	fetchGroup           singleflight.Group
	segmentCache         SegmentCache
}

func NewPool(conf *Config) (*Pool, error) {
	conf.setDefaults()

	up := &Pool{
		Log:                  conf.Log,
		providers:            []*providerPool{},
		requiredCapabilities: conf.RequiredCapabilities,
		minConnections:       conf.MinConnections,
		segmentCache:         conf.SegmentCache,
	}

	for i := range conf.Providers {
		provider := &conf.Providers[i]
		err := up.addProvider(provider)
		if err != nil {
			return nil, err
		}
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
		c, err := p.GetConnection(ctx, nil, 9, false)
		if err != nil {
			return err
		}
		c.Release()
	}
	return nil
}

func (p *Pool) verifyProvider(provider *providerPool) {
	c, err := provider.Acquire(context.Background())
	if err != nil {
		p.Log.Error("marking provider pool offline due to failed connection test", "error", err, "id", provider.Id())
		provider.SetState(nntp.PoolStateOffline)
		return
	}
	defer c.Release()

	if len(p.requiredCapabilities) > 0 {
		caps, err := c.Capabilities()
		if err != nil {
			p.Log.Error("marking provider pool offline due to failed capabilities test", "error", err, "id", provider.Id())
			provider.SetState(nntp.PoolStateOffline)
			return
		}

		for _, capability := range p.requiredCapabilities {
			if !slices.Contains(caps.Capabilities, capability) {
				p.Log.Warn("marking provider pool disabled due to missing required capability", "capability", capability, "id", provider.Id())
				provider.SetState(nntp.PoolStateDisabled)
				return
			}
		}
	}
}

func (p *Pool) verifyProviders() {
	var wg sync.WaitGroup
	for _, provider := range p.providers {
		wg.Go(func() {
			p.verifyProvider(provider)
		})
	}
	wg.Wait()
}

func (p *Pool) GetConnection(ctx context.Context, excludeProvider []string, maxPriority int, useBackup bool) (*nntp.PooledConnection, error) {
	p.providersMutex.RLock()
	if len(p.providers) == 0 {
		p.providersMutex.RUnlock()
		return nil, ErrNoProvidersAvailable
	}
	providers := make([]*providerPool, 0, len(p.providers))
	for _, provider := range p.providers {
		if !provider.IsOnline() {
			continue
		}
		if provider.priority > maxPriority {
			continue
		}
		if provider.isBackup != useBackup {
			continue
		}
		if slices.Contains(excludeProvider, provider.Id()) {
			continue
		}
		providers = append(providers, provider)
	}
	p.providersMutex.RUnlock()

	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	for _, provider := range providers {
		if provider.Stat().AcquiredResources() == provider.MaxSize() {
			continue
		}
		conn, err := provider.Acquire(ctx)
		if err == nil {
			return conn, nil
		}
		p.Log.Debug("failed to acquire connection from provider", "error", err, "provider_id", provider.Id())
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

func (p *Pool) fetchSegment(ctx context.Context, segment *nzb.Segment, groups []string) (*SegmentData, error) {
	messageId := segment.MessageId
	if cachedData, ok := p.segmentCache.Get(messageId); ok {
		p.Log.Trace("fetch segment - cache hit", "segment_num", segment.Number, "message_id", messageId, "size", len(cachedData.Body))
		return &cachedData, nil
	}

	result, err, _ := p.fetchGroup.Do(messageId, func() (any, error) {
		var excludeProviders []string
		errs := []error{}
		failedAttempts := 0
		currPriority := 0
		useBackup := false

		for failedAttempts < 3 {
			if len(excludeProviders) > 0 || currPriority > 0 || useBackup {
				p.Log.Trace("fetch segment - retry", "segment_num", segment.Number, "message_id", messageId, "failed_attempts", failedAttempts, "excluded_providers", len(excludeProviders), "curr_priority", currPriority, "use_backup", useBackup)
			}

			conn, err := p.GetConnection(context.Background(), excludeProviders, currPriority, useBackup)
			if err != nil {
				if errors.Is(err, ErrNoProvidersAvailable) {
					if currPriority < 9 {
						currPriority++
						p.Log.Trace("fetch segment - expanding to lower priority", "segment_num", segment.Number, "message_id", messageId, "new_priority", currPriority, "use_backup", useBackup)
						continue
					}
					if !useBackup && len(excludeProviders) > 0 {
						useBackup = true
						currPriority = 0
						p.Log.Trace("fetch segment - switching to backup providers", "segment_num", segment.Number, "message_id", messageId)
						continue
					}
					errs = append(errs, err)
					break
				}
				errs = append(errs, err)
				failedAttempts++
				p.Log.Warn("fetch segment - failed to get connection", "error", err, "segment_num", segment.Number, "message_id", messageId)
				continue
			}

			p.Log.Trace("fetch segment - connection acquired", "segment_num", segment.Number, "message_id", messageId, "provider_id", conn.ProviderId(), "use_backup", useBackup)

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

			segmentData := data.ToSegmentData()

			p.Log.Debug("fetch segment - decoded body", "segment_num", segment.Number, "message_id", messageId, "decoded_size", len(segmentData.Body))

			p.segmentCache.Set(messageId, segmentData)

			return &segmentData, nil
		}

		allArticleNotFound := len(errs) > 0
		for _, e := range errs {
			if e != nil && !isArticleNotFoundError(e) {
				allArticleNotFound = false
				break
			}
		}
		retryErr := errors.Join(errs...)
		if allArticleNotFound {
			return nil, fmt.Errorf("%w: failed to fetch segment %d <%s> after retries: %s", ErrArticleNotFound, segment.Number, messageId, retryErr.Error())
		}
		return nil, errors.New("failed to fetch segment " + strconv.Itoa(segment.Number) + " <" + messageId + "> after retries: " + retryErr.Error())
	})

	if err != nil {
		return nil, err
	}

	return result.(*SegmentData), nil
}

func (p *Pool) Close() {
	p.providersMutex.Lock()
	defer p.providersMutex.Unlock()

	for _, provider := range p.providers {
		provider.Close()
	}
}

func (p *Pool) addProvider(provider *ProviderConfig) error {
	if provider.Log == nil {
		provider.Log = p.Log.With("id", provider.Id())
	}

	pool, err := nntp.NewPool(&provider.PoolConfig)
	if err != nil {
		return err
	}

	pPool := &providerPool{
		Pool:     pool,
		priority: provider.Priority,
		isBackup: provider.IsBackup,
	}

	p.verifyProvider(pPool)

	p.providers = append(p.providers, pPool)

	return nil
}

func (p *Pool) AddProvider(provider *ProviderConfig) error {
	p.providersMutex.Lock()
	defer p.providersMutex.Unlock()

	p.addProvider(provider)
	p.Log.Info("provider added", "id", provider.Id())
	return nil
}

func (p *Pool) RemoveProvider(serverId string) {
	p.providersMutex.Lock()
	defer p.providersMutex.Unlock()

	for i, provider := range p.providers {
		if provider.Id() == serverId {
			provider.Close()
			p.providers = slices.Delete(p.providers, i, i+1)
			p.Log.Info("provider removed", "id", serverId)
			return
		}
	}
}

func (p *Pool) CountProviders() int {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()
	return len(p.providers)
}

func (p *Pool) HasProvider(serverId string) bool {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()

	for _, provider := range p.providers {
		if provider.Id() == serverId {
			return true
		}
	}
	return false
}

func (p *Pool) HasActiveConnections() bool {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()

	for _, provider := range p.providers {
		if provider.Stat().AcquiredResources() > 0 {
			return true
		}
	}
	return false
}

func (p *Pool) GetAcquiredConnectionCount(providerId string) int {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()

	for _, provider := range p.providers {
		if provider.Id() == providerId {
			return int(provider.Stat().AcquiredResources())
		}
	}
	return 0
}

type ProviderInfo struct {
	ID                string         `json:"id"`
	State             nntp.PoolState `json:"state"`
	Priority          int            `json:"priority"`
	IsBackup          bool           `json:"is_backup"`
	MaxConnections    int            `json:"max_connections"`
	TotalConnections  int            `json:"total_connections"`
	ActiveConnections int            `json:"active_connections"`
	IdleConnections   int            `json:"idle_connections"`
}

type PoolInfo struct {
	TotalProviders    int            `json:"total_providers"`
	MaxConnections    int            `json:"max_connections"`
	ActiveConnections int            `json:"active_connections"`
	IdleConnections   int            `json:"idle_connections"`
	Providers         []ProviderInfo `json:"providers"`
}

func (p *Pool) GetPoolInfo() PoolInfo {
	p.providersMutex.RLock()
	defer p.providersMutex.RUnlock()

	info := PoolInfo{
		Providers: make([]ProviderInfo, 0, len(p.providers)),
	}

	for _, provider := range p.providers {
		stat := provider.Stat()

		pi := ProviderInfo{
			ID:                provider.Id(),
			State:             provider.GetState(),
			Priority:          provider.priority,
			IsBackup:          provider.isBackup,
			MaxConnections:    int(provider.MaxSize()),
			TotalConnections:  int(stat.TotalResources()),
			ActiveConnections: int(stat.AcquiredResources()),
			IdleConnections:   int(stat.IdleResources()),
		}

		if provider.IsOnline() {
			info.MaxConnections += pi.MaxConnections
			info.ActiveConnections += pi.ActiveConnections
			info.IdleConnections += pi.IdleConnections
		}
		info.Providers = append(info.Providers, pi)
	}

	info.TotalProviders = len(info.Providers)

	return info
}
