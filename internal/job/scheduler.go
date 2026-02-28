package job

import (
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

type stoppable interface {
	init()
	stop()
}

type Scheduler[T any] struct {
	conf       *SchedulerConfig[T]
	jobTracker *JobTracker[T]
	mu         sync.Mutex
	scheduler  *tasks.Scheduler
	triggerCh  chan struct{}
}

func (sch *Scheduler[T]) Logger() *logger.Logger {
	return sch.conf.Log
}

func (sch *Scheduler[T]) JobQueue() job_queue.JobQueue[T] {
	return sch.conf.Queue
}

func (sch *Scheduler[T]) init() {
	conf := sch.conf
	log := conf.Log

	jobTrackerExpiresIn := max(3*24*time.Hour, 10*conf.Interval)
	sch.jobTracker = NewJobTracker[T](conf.Id, jobTrackerExpiresIn)

	if conf.Interval > 0 {
		sch.scheduler = tasks.New()

		id, err := sch.scheduler.Add(&tasks.Task{
			Interval:          conf.Interval,
			RunSingleInstance: true,
			TaskFunc: func() error {
				sch.execute(false)
				return nil
			},
		})

		if err != nil {
			panic(err)
		}

		log.Info("Started Job Scheduler", "id", id)
	} else {
		log.Info("Started Job Scheduler (trigger-only)")
	}

	if conf.RunAtStartupAfter != 0 {
		time.AfterFunc(conf.RunAtStartupAfter, func() {
			sch.execute(false)
		})
	}

	go func() {
		for range sch.triggerCh {
			sch.execute(true)
		}
	}()
}

func (j *Scheduler[T]) stop() {
	close(j.triggerCh)
	if j.scheduler != nil {
		j.scheduler.Stop()
	}
}

type SchedulerConfig[T any] struct {
	Disabled          bool
	Executor          func(j *Scheduler[T]) error
	HeartbeatInterval time.Duration
	Interval          time.Duration
	Log               *logger.Logger
	Id                string
	Title             string
	OnEnd             func()
	OnStart           func()
	Queue             job_queue.JobQueue[T]
	RunAtStartupAfter time.Duration
	RunExclusive      bool
	ShouldSkip        func() bool
	ShouldWait        func() (bool, string)
}

type JobDetail struct {
	Id       string        `json:"id"`
	Title    string        `json:"title"`
	Interval time.Duration `json:"interval"`
}

const (
	JobStatusStarted = "started"
	JobStatusDone    = "done"
	JobStatusFailed  = "failed"
)

func (j *Scheduler[T]) Trigger(payload T) error {
	if err := j.JobQueue().Queue(payload, 1); err != nil {
		return err
	}
	select {
	case j.triggerCh <- struct{}{}:
	default:
	}
	return nil
}

func (j *Scheduler[T]) execute(triggered bool) {
	j.mu.Lock()
	defer j.mu.Unlock()

	conf := j.conf

	log := conf.Log

	if conf.ShouldSkip() {
		log.Trace("skipping")
		return
	}

	for {
		wait, reason := conf.ShouldWait()
		if !wait {
			break
		}
		log.Info("waiting, " + reason)
		time.Sleep(1 * time.Minute)
	}

	lock := db.NewAdvisoryLock("job", conf.Id)
	if lock == nil {
		log.Error("failed to create advisory lock", "name", conf.Id)
		return
	}

	if !lock.TryAcquire() {
		log.Debug("skipping, another instance is running", "name", lock.GetName())
		return
	}
	defer lock.Release()

	conf.OnStart()
	defer func() {
		if perr, stack := util.HandlePanic(recover(), true); perr != nil {
			log.Error("Job Panic", "error", perr, "stack", stack)
		}
		conf.OnEnd()
	}()

	jobTracker := j.jobTracker
	heartbeatIntervalTolerance := min(conf.HeartbeatInterval, 10*time.Second)

	if conf.RunExclusive {
		tjob, err := jobTracker.GetLast()
		if err != nil {
			log.Error("failed to get last job", "error", err)
			return
		}
		if tjob != nil {
			status := tjob.Status
			switch status {
			case JobStatusStarted:
				if !util.HasDurationPassedSince(tjob.UpdatedAt, conf.HeartbeatInterval+heartbeatIntervalTolerance) {
					if util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
						log.Warn("skipping, last job is still running, for too long", "jobId", tjob.Id, "status", status)
					} else {
						log.Info("skipping, last job is still running", "jobId", tjob.Id, "status", status)
					}
					return
				}

				log.Warn("last job heartbeat timed out, restarting", "jobId", tjob.Id, "status", status)
				if err := jobTracker.Set(tjob.Id, JobStatusFailed, "heartbeat timed out", nil); err != nil {
					log.Error("failed to set last job status", "error", err, "jobId", tjob.Id, "status", JobStatusFailed)
				}
			case JobStatusDone:
				if !triggered && !util.HasDurationPassedSince(tjob.CreatedAt, conf.Interval) {
					log.Info("already done", "jobId", tjob.Id, "status", status)
					return
				}
			case JobStatusFailed:
				log.Warn("last job failed", "jobId", tjob.Id, "status", status, "error", tjob.Error)
			}
		}
	}

	jobId := time.Now().Format(time.DateTime)

	if err := jobTracker.Set(jobId, JobStatusStarted, "", nil); err != nil {
		log.Error("failed to set job status", "error", err, "jobId", jobId, "status", JobStatusStarted)
		return
	}

	if !lock.Release() {
		log.Error("failed to release advisory lock", "name", lock.GetName())
		return
	}

	heartbeat := time.NewTicker(conf.HeartbeatInterval)
	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go func() {
		for {
			select {
			case <-heartbeat.C:
				if err := jobTracker.Set(jobId, JobStatusStarted, "", nil); err != nil {
					log.Error("failed to set job status heartbeat", "error", err, "jobId", jobId)
				}
			case <-heartbeatDone:
				heartbeat.Stop()
				return
			}
		}
	}()

	if err := conf.Executor(j); err != nil {
		log.Error("Job Failure", "error", err)
		if terr := jobTracker.Set(jobId, JobStatusFailed, err.Error(), nil); terr != nil {
			log.Error("failed to set job status", "error", terr, "jobId", jobId, "status", JobStatusFailed)
		}
		return
	}

	if err := jobTracker.Set(jobId, JobStatusDone, "", nil); err != nil {
		log.Error("failed to set job status", "error", err, "jobId", jobId, "status", JobStatusDone)
		return
	}

	log.Info("done", "jobId", jobId)
}

func NewScheduler[T any](conf *SchedulerConfig[T]) *Scheduler[T] {
	if conf.Id == "" {
		panic("scheduler id cannot be empty")
	}

	if _, ok := JobDetailsById[conf.Id]; ok {
		panic("scheduler already registered: " + conf.Id)
	} else {
		JobDetailsById[conf.Id] = &JobDetail{
			Id:       conf.Id,
			Interval: conf.Interval,
			Title:    conf.Title,
		}
	}

	if conf.Disabled {
		return nil
	}

	if conf.Log == nil {
		conf.Log = logger.Scoped("job/" + conf.Id)
	}

	if conf.HeartbeatInterval == 0 {
		conf.HeartbeatInterval = 5 * time.Second
	}

	if conf.OnStart == nil {
		conf.OnStart = func() {}
	}
	if conf.OnEnd == nil {
		conf.OnEnd = func() {}
	}
	if conf.ShouldSkip == nil {
		conf.ShouldSkip = func() bool {
			return false
		}
	}
	if conf.ShouldWait == nil {
		conf.ShouldWait = func() (bool, string) {
			return false, ""
		}
	}

	if conf.Queue == nil {
		conf.Queue = job_queue.NewMemoryJobQueue(job_queue.JobQueueConfig[T]{
			GetKey: func(item *T) string { return "" },
		})
	}

	sch := &Scheduler[T]{
		triggerCh: make(chan struct{}, 1),
		conf:      conf,
	}

	registerJob(conf.Id, sch)

	return sch
}
