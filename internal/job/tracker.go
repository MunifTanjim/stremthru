package job

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/job_log"
)

type JobLog[T any] = job_log.ParsedJobLog[T]

type JobTracker[T any] struct {
	name      string
	expiresIn time.Duration
}

func (t JobTracker[T]) Get(id string) (*JobLog[T], error) {
	if id == "" {
		panic("job id cannot be empty")
	}
	jl, err := job_log.GetJobLog[T](t.name, id)
	if err != nil {
		return nil, err
	}
	if jl == nil {
		return &JobLog[T]{}, nil
	}
	return jl, nil
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
	return j.Status == JobStatusStarted, nil
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
