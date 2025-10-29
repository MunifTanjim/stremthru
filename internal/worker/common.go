package worker

import (
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/job_log"
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
	name      string
	expiresIn time.Duration
}

func (t JobTracker[T]) Get(id string) (*Job[T], error) {
	if id == "" {
		panic("job id cannot be empty")
	}
	pjl, err := job_log.GetJobLog[T](t.name, id)
	if err != nil {
		return nil, err
	}
	if pjl == nil {
		return &Job[T]{}, nil
	}
	return &Job[T]{
		Status: pjl.Status,
		Err:    pjl.Error,
		Data:   pjl.Data,
	}, nil
}

func (t JobTracker[T]) GetLast() (*job_log.ParsedJobLog[T], error) {
	pjl, err := job_log.GetLastJobLog[T](t.name)
	if err != nil {
		return nil, err
	}
	if pjl == nil {
		return nil, nil
	}
	if pjl.Id == "" {
		return nil, job_log.DeleteJobLog(t.name, pjl.Id)
	}
	return pjl, nil
}

func (t JobTracker[T]) Set(id string, status string, err string, data *T) error {
	if id == "" {
		panic("job id cannot be empty")
	}
	return job_log.SaveJobLog(t.name, id, status, data, err, t.expiresIn)
}

func (t JobTracker[T]) IsRunning(id string) (bool, error) {
	j, err := t.Get(id)
	if err != nil {
		return false, err
	}
	return j.Status == "started", nil
}

func NewJobTracker[T any](name string, expiresIn time.Duration) *JobTracker[T] {
	tracker := JobTracker[T]{
		name:      name,
		expiresIn: expiresIn,
	}
	if _, err := job_log.GetAllJobLogs[T](name); err != nil {
		panic(err)
	}
	return &tracker
}
