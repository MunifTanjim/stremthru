package worker_queue

import (
	"errors"
	"sync"
	"time"
)

type WorkerQueueItem[T any] struct {
	v T
	t time.Time
}

type WorkerQueue[T any] struct {
	m            sync.Map
	getKey       func(item T) string
	getGroupKey  func(item T) string
	transform    func(item *T) *T
	debounceTime time.Duration
	Disabled     bool
}

var ErrWorkerQueueItemDelayed = errors.New("worker queue item delayed")

func (q *WorkerQueue[T]) Queue(item T) {
	if q.Disabled {
		return
	}
	item = *q.transform(&item)
	q.m.Swap(q.getKey(item), WorkerQueueItem[T]{
		v: item,
		t: time.Now().Add(q.debounceTime),
	})
}

func (q *WorkerQueue[T]) delete(item T) {
	q.m.Delete(q.getKey(item))
}

func (q *WorkerQueue[T]) IsEmpty() bool {
	isEmpty := true
	q.m.Range(func(k, v any) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}

func (q *WorkerQueue[T]) Process(f func(item T) error) {
	q.m.Range(func(k, v any) bool {
		_, keyOk := k.(string)
		val, valOk := v.(WorkerQueueItem[T])
		if keyOk && valOk && val.t.Before(time.Now()) {
			if err := f(val.v); err != nil {
				if err == ErrWorkerQueueItemDelayed {
					log.Debug("WorkerQueue process delayed", "key", q.getKey(val.v))
				} else {
					log.Error("WorkerQueue process failed", "error", err, "key", q.getKey(val.v))
				}
			} else {
				q.delete(val.v)
			}
		}
		return true
	})
}

func (q *WorkerQueue[T]) ProcessGroup(f func(groupKey string, items []T) error) {
	byGroupKey := map[string][]T{}
	q.m.Range(func(k, v any) bool {
		_, keyOk := k.(string)
		val, valOk := v.(WorkerQueueItem[T])
		if keyOk && valOk && val.t.Before(time.Now()) {
			groupKey := q.getGroupKey(val.v)
			if _, ok := byGroupKey[groupKey]; !ok {
				byGroupKey[groupKey] = []T{}
			}
			byGroupKey[groupKey] = append(byGroupKey[groupKey], val.v)
		}
		return true
	})
	for groupKey, items := range byGroupKey {
		if err := f(groupKey, items); err != nil {
			if err == ErrWorkerQueueItemDelayed {
				log.Debug("WorkerQueue processGroup delayed", "group_key", groupKey)
			} else {
				log.Error("WorkerQueue processGroup failed", "error", err, "group_key", groupKey)
			}
		} else {
			for i := range items {
				q.delete(items[i])
			}
		}
	}
}
