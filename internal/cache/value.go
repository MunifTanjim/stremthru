package cache

import (
	"time"

	"golang.org/x/sync/singleflight"
)

type CachedValue[T any] struct {
	g       singleflight.Group
	get     func() (T, error)
	staleAt time.Time
	ttl     time.Duration
	value   T
}

func (cv *CachedValue[T]) Get() (T, error) {
	if cv.staleAt.Before(time.Now()) {
		_, err, _ := cv.g.Do("", func() (any, error) {
			v, err := cv.get()
			if err != nil {
				return nil, err
			}
			cv.value = v
			cv.staleAt = time.Now().Add(cv.ttl)
			return nil, nil
		})
		if err != nil {
			return cv.value, err
		}
	}
	return cv.value, nil
}

func (cv *CachedValue[T]) Invalidate() {
	cv.staleAt = time.Unix(0, 0)
}

func (cv *CachedValue[T]) StaleAt() time.Time {
	return cv.staleAt
}

type CachedValueConfig[T any] struct {
	Get func() (T, error)
	TTL time.Duration
}

func NewCachedValue[T any](conf CachedValueConfig[T]) *CachedValue[T] {
	return &CachedValue[T]{
		get:     conf.Get,
		staleAt: time.Unix(0, 0),
		ttl:     conf.TTL,
	}
}
