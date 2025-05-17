package worker

import (
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/kv"
)

type IdQueue struct {
	m            sync.Map
	debounceTime time.Duration
	transform    func(id string) string
	disabled     bool
}

func (q *IdQueue) Queue(sid string) {
	if q.disabled {
		return
	}
	q.m.Swap(q.transform(sid), time.Now().Add(q.debounceTime))
}

func (q *IdQueue) delete(sid string) {
	q.m.Delete(q.transform(sid))
}

type WorkerQueueItem[T any] struct {
	v T
	t time.Time
}

type WorkerQueue[T any] struct {
	m            sync.Map
	getKey       func(item T) string
	transform    func(item *T) *T
	debounceTime time.Duration
}

func (q *WorkerQueue[T]) Queue(item T) {
	item = *q.transform(&item)
	q.m.Swap(q.getKey(item), WorkerQueueItem[T]{
		v: item,
		t: time.Now().Add(q.debounceTime),
	})
}

func (q *WorkerQueue[T]) delete(item T) {
	q.m.Delete(q.getKey(item))
}

func (q *WorkerQueue[T]) process(f func(item T)) {
	q.m.Range(func(k, v any) bool {
		_, keyOk := k.(string)
		val, valOk := v.(WorkerQueueItem[T])
		if keyOk && valOk && val.t.Before(time.Now()) {
			f(val.v)
			q.delete(val.v)
		}
		return true
	})
}

type Error struct {
	string
	cause error
}

func (e Error) Error() string {
	return e.string + "\n" + e.cause.Error()
}

type Job[T any] struct {
	Status string `json:"status"`
	Err    string `json:"err"`
	Data   *T     `json:"data,omitempty"`
}

type JobTracker[T any] struct {
	kv kv.KVStore[Job[T]]
}

func (t JobTracker[T]) Get(id string) (*Job[T], error) {
	j := Job[T]{}
	err := t.kv.Get(id, &j)
	return &j, err
}

func (t JobTracker[T]) cleanup(fn func(id string, j *Job[T]) bool) error {
	items, err := t.kv.List()
	if err != nil {
		return err
	}
	for i := range items {
		if fn(items[i].Key, &items[i].Value) {
			err := t.kv.Del(items[i].Key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t JobTracker[T]) Set(id string, status string, err string, data *T) error {
	terr := t.kv.Set(id, Job[T]{
		Status: status,
		Err:    err,
		Data:   data,
	})
	return terr
}

func (t JobTracker[T]) IsRunning(id string) (bool, error) {
	j, err := t.Get(id)
	return j.Status == "started", err
}

func NewJobTracker[T any](name string, shouldClean func(id string, j *Job[T]) bool) JobTracker[T] {
	tracker := JobTracker[T]{
		kv: kv.NewKVStore[Job[T]](&kv.KVStoreConfig{
			Type: "job:" + name,
		}),
	}
	err := tracker.cleanup(shouldClean)
	if err != nil {
		panic(err)
	}
	return tracker
}
