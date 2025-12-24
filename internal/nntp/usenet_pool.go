package nntp

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

type ProviderConfig = PoolConfig

type UsenetProviderConfig struct {
	ProviderConfig
	IsBackup bool
}

type UsenetPoolConfig struct {
	Log                  *logger.Logger
	Providers            []UsenetProviderConfig
	RequiredCapabilities []string
	MinConnections       int
}

func (upc *UsenetPoolConfig) setDefaults() {
	if upc.Log == nil {
		upc.Log = logger.Scoped("nntp/usenet-pool")
	}
	slices.SortStableFunc(upc.Providers, func(a, b UsenetProviderConfig) int {
		if a.IsBackup && !b.IsBackup {
			return 1
		}
		if !a.IsBackup && b.IsBackup {
			return -1
		}
		return 0
	})
}

type providerPool struct {
	*Pool
	isBackup bool
}

type UsenetPool struct {
	Log                  *logger.Logger
	providers            []*providerPool
	providersMutex       sync.RWMutex
	requiredCapabilities []string
	minConnections       int
}

func NewUsenetPool(config *UsenetPoolConfig) (*UsenetPool, error) {
	config.setDefaults()

	pools := []*providerPool{}

	for i := range config.Providers {
		provider := &config.Providers[i]
		if provider.ProviderConfig.Log == nil {
			provider.ProviderConfig.Log = config.Log.With("id", provider.ProviderConfig.getId())
		}
		pool, err := NewPool(&provider.ProviderConfig)
		if err != nil {
			return nil, err
		}
		pools = append(pools, &providerPool{
			Pool:     pool,
			isBackup: provider.IsBackup,
		})
	}

	up := &UsenetPool{
		Log:                  config.Log,
		providers:            pools,
		requiredCapabilities: config.RequiredCapabilities,
		minConnections:       config.MinConnections,
	}

	up.verifyProviders()

	if err := up.ensureMinSize(context.Background()); err != nil {
		up.Log.Warn("failed to ensure min size at startup", "error", err)
	}

	return up, nil
}

func (up *UsenetPool) ensureMinSize(ctx context.Context) error {
	if up.minConnections == 0 {
		return nil
	}

	currentCount := 0
	for _, provider := range up.providers {
		if provider.IsOnline() {
			currentCount += int(provider.Stat().TotalResources())
		}
	}

	if currentCount >= up.minConnections {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for range up.minConnections - currentCount {
		c, err := up.GetConnection(ctx, nil, false)
		if err != nil {
			return err
		}
		c.Release()
	}
	return nil
}

func (up *UsenetPool) verifyProviders() {
	if len(up.requiredCapabilities) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, provider := range up.providers {
		wg.Go(func() {
			c, err := provider.Acquire(context.Background())
			if err != nil {
				return
			}
			defer c.Release()

			caps, err := c.Capabilities()
			if err != nil {
				provider.SetState(PoolStateOffline)
				return
			}

			for _, capability := range up.requiredCapabilities {
				if !slices.Contains(caps.Capabilities, capability) {
					provider.SetState(PoolStateDisabled)
					up.Log.Warn("disabling provider pool due to missing required capability", "capability", capability, "host", provider.config.Host)
					return
				}
			}
		})
	}
	wg.Wait()
}

func (up *UsenetPool) GetConnection(ctx context.Context, excludeProvider []string, includeBackup bool) (*PooledConnection, error) {
	up.providersMutex.RLock()
	if len(up.providers) == 0 {
		up.providersMutex.RUnlock()
		return nil, errors.New("usenet: no providers configured")
	}
	providers := make([]*providerPool, 0, len(up.providers))
	for _, provider := range up.providers {
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
	up.providersMutex.RUnlock()

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

func (up *UsenetPool) Close() {
	up.providersMutex.Lock()
	defer up.providersMutex.Unlock()

	for _, provider := range up.providers {
		provider.Close()
	}
}

func (up *UsenetPool) StreamFile(ctx context.Context, config StreamFileConfig) (*StreamFileResult, error) {
	if len(config.Segments) == 0 {
		return nil, errors.New("no segments provided")
	}

	// Set defaults
	parallelism := config.ParallelDownloads
	if parallelism <= 0 {
		parallelism = defaultParallelDownloads
	}

	bufferAhead := config.BufferAhead
	if bufferAhead <= 0 {
		bufferAhead = defaultBufferAhead
	}

	up.Log.Trace("initializing file stream", "segments", len(config.Segments), "parallelism", parallelism, "buffer_ahead", bufferAhead)

	// Create cancelable context
	ctx, cancel := context.WithCancel(ctx)

	reader := &UsenetFileReader{
		pool:        up,
		ctx:         ctx,
		cancel:      cancel,
		segments:    config.Segments,
		groups:      config.Groups,
		parallelism: parallelism,
		cache:       newSegmentCache(parallelism + bufferAhead),
		log:         up.Log,
	}

	// Calculate byte offsets
	reader.calculateOffsets()

	up.Log.Trace("file stream ready", "total_bytes", reader.totalSize, "cache_size", parallelism+bufferAhead)

	// Start prefetch workers
	reader.startWorkers()

	return &StreamFileResult{
		ReadSeekCloser: reader,
		Size:           reader.totalSize,
	}, nil
}
