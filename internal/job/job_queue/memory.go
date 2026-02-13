package job_queue

import (
	"errors"
	"slices"
	"sync"
	"time"
)

type jobQueueItem[T any] struct {
	v T
	t time.Time
	p int
}

type MemoryJobQueue[T any] struct {
	m            sync.Map
	getKey       func(item *T) string
	getGroupKey  func(item *T) string
	debounceTime time.Duration
	disabled     bool
}

func (q *MemoryJobQueue[T]) Queue(item T, priority ...int) error {
	if q.disabled {
		return nil
	}
	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}
	key := q.getKey(&item)
	if existing, loaded := q.m.Load(key); loaded {
		if ev, ok := existing.(jobQueueItem[T]); ok && ev.p > p {
			p = ev.p
		}
	}
	q.m.Store(key, jobQueueItem[T]{
		v: item,
		t: time.Now().Add(q.debounceTime),
		p: p,
	})
	return nil
}

func (q *MemoryJobQueue[T]) delete(item *T) {
	q.m.Delete(q.getKey(item))
}

func (q *MemoryJobQueue[T]) IsEmpty() bool {
	isEmpty := true
	q.m.Range(func(k, v any) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}

func (q *MemoryJobQueue[T]) Process(f func(item T) error) {
	type readyItem struct {
		key string
		val jobQueueItem[T]
	}
	for {
		var items []readyItem
		now := time.Now()
		q.m.Range(func(k, v any) bool {
			key, keyOk := k.(string)
			val, valOk := v.(jobQueueItem[T])
			if keyOk && valOk && val.t.Before(now) {
				items = append(items, readyItem{key: key, val: val})
			}
			return true
		})
		if len(items) == 0 {
			return
		}
		slices.SortStableFunc(items, func(a, b readyItem) int {
			return b.val.p - a.val.p
		})
		for _, item := range items {
			if err := f(item.val.v); err != nil {
				var delayed *ErrJobQueueItemDelayed
				if errors.As(err, &delayed) {
					q.m.Store(item.key, jobQueueItem[T]{v: item.val.v, t: time.Now().Add(delayed.RetryAfter), p: item.val.p})
					log.Debug("JobQueue process delayed", "key", item.key, "retry_after", delayed.RetryAfter)
				} else {
					log.Error("JobQueue process failed", "error", err, "key", item.key)
					return
				}
			} else {
				q.delete(&item.val.v)
			}
		}
	}
}

func (q *MemoryJobQueue[T]) ProcessGroup(f func(groupKey string, items []T) error) {
	type readyItem struct {
		key string
		val jobQueueItem[T]
	}
	for {
		byGroupKey := map[string][]readyItem{}
		now := time.Now()
		q.m.Range(func(k, v any) bool {
			key, keyOk := k.(string)
			val, valOk := v.(jobQueueItem[T])
			if keyOk && valOk && val.t.Before(now) {
				groupKey := q.getGroupKey(&val.v)
				byGroupKey[groupKey] = append(byGroupKey[groupKey], readyItem{key: key, val: val})
			}
			return true
		})
		if len(byGroupKey) == 0 {
			return
		}
		for groupKey, readyItems := range byGroupKey {
			items := make([]T, len(readyItems))
			for i, ri := range readyItems {
				items[i] = ri.val.v
			}
			if err := f(groupKey, items); err != nil {
				var delayed *ErrJobQueueItemDelayed
				if errors.As(err, &delayed) {
					for _, ri := range readyItems {
						q.m.Store(ri.key, jobQueueItem[T]{v: ri.val.v, t: time.Now().Add(delayed.RetryAfter), p: ri.val.p})
					}
					log.Debug("JobQueue process group delayed", "group_key", groupKey, "retry_after", delayed.RetryAfter)
				} else {
					log.Error("JobQueue process group failed", "error", err, "group_key", groupKey)
					return
				}
			} else {
				for _, ri := range readyItems {
					q.delete(&ri.val.v)
				}
			}
		}
	}
}
