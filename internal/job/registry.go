package job

import (
	"fmt"
	"sync"
)

var JobDetailsById = map[string]*JobDetail{}

var jobsByName sync.Map

func registerJob[T any](name string, j *Scheduler[T]) {
	jobsByName.Store(name, j)
}

func Trigger[T any](name string, payload T) error {
	v, ok := jobsByName.Load(name)
	if !ok {
		return fmt.Errorf("job not found: %s", name)
	}
	return v.(*Scheduler[T]).Trigger(payload)
}

func GetJobTracker[T any](name string) *JobTracker[T] {
	v, ok := jobsByName.Load(name)
	if !ok {
		return nil
	}
	return v.(*Scheduler[T]).jobTracker
}

func InitJobs() func() {
	jobsByName.Range(func(key, value any) bool {
		if j, ok := value.(stoppable); ok {
			j.init()
		}
		return true
	})
	return func() {
		jobsByName.Range(func(key, value any) bool {
			if j, ok := value.(stoppable); ok {
				j.stop()
			}
			return true
		})
	}
}
