package job_queue

import (
	"time"
)

type JobQueue[T any] interface {
	Queue(item T, priority ...int) error
	IsEmpty() bool
	Process(f func(item T) error)
	ProcessGroup(f func(groupKey string, items []T) error)
}

type ErrJobQueueItemDelayed struct {
	RetryAfter time.Duration
}

func (e *ErrJobQueueItemDelayed) Error() string {
	return "job queue item delayed"
}

type JobQueueConfig[T any] struct {
	GetKey       func(item *T) string
	GetGroupKey  func(item *T) string
	DebounceTime time.Duration
	BackoffDelay time.Duration
	MaxRetry     int
	Disabled     bool
}

func (conf *JobQueueConfig[T]) setDefaults() {
	if conf.GetKey == nil {
		panic("GetKey function is required")
	}
	if conf.GetGroupKey == nil {
		conf.GetGroupKey = conf.GetKey
	}
	if conf.BackoffDelay == 0 {
		conf.BackoffDelay = time.Minute
	}
}

func NewMemoryJobQueue[T any](conf JobQueueConfig[T]) JobQueue[T] {
	conf.setDefaults()

	return &MemoryJobQueue[T]{
		getKey:       conf.GetKey,
		getGroupKey:  conf.GetGroupKey,
		debounceTime: conf.DebounceTime,
		disabled:     conf.Disabled,
	}
}

func NewPersistentJobQueue[T any](name string, conf JobQueueConfig[T]) JobQueue[T] {
	conf.setDefaults()

	return &PersistentJobQueue[T]{
		name:         name,
		getKey:       conf.GetKey,
		getGroupKey:  conf.GetGroupKey,
		debounceTime: conf.DebounceTime,
		backoffDelay: conf.BackoffDelay,
		maxRetry:     conf.MaxRetry,
		disabled:     conf.Disabled,
	}
}
