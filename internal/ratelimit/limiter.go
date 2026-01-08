package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/redis"
	rrl "github.com/nccapo/rate-limiter"
)

var cachedLimiterById sync.Map // map[string]*Limiter

func createStore() rrl.Store {
	if redis.IsAvailable() {
		return rrl.NewRedisStore(redis.GetClient(), true)
	}
	return rrl.NewMemoryStore()
}

type Limiter struct {
	config *RateLimitConfig
	rl     *rrl.RateLimiter
}

func (l *Limiter) Try(key string) (*rrl.RateLimitResult, error) {
	return l.rl.Allow(context.Background(), key)
}

func (l *Limiter) Wait(key string) error {
	return l.rl.Wait(context.Background(), key)
}

func (l *Limiter) Config() *RateLimitConfig {
	return l.config
}

func NewLimiter(conf *RateLimitConfig) (*Limiter, error) {
	window, err := conf.ParseWindow()
	if err != nil {
		return nil, fmt.Errorf("failed to parse window: %w", err)
	}

	refillInterval := window / time.Duration(conf.Limit)

	limiter, err := rrl.NewRateLimiter(
		rrl.WithMaxTokens(int64(conf.Limit)),
		rrl.WithRefillInterval(refillInterval),
		rrl.WithStore(createStore()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate limiter: %w", err)
	}

	return &Limiter{
		config: conf,
		rl:     limiter,
	}, nil
}

func NewLimiterById(id string) (*Limiter, error) {
	if cached, ok := cachedLimiterById.Load(id); ok {
		return cached.(*Limiter), nil
	}

	cfg, err := GetById(id)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("rate limit config not found: %s", id)
	}

	limiter, err := NewLimiter(cfg)
	if err != nil {
		return nil, err
	}

	cachedLimiterById.Store(id, limiter)
	return limiter, nil
}
