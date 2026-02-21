package torznab_indexer_syncinfo

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
)

type JobData struct {
	SId string `json:"sid"`
}

var queue = job_queue.NewMemoryJobQueue(job_queue.JobQueueConfig[JobData]{
	DebounceTime: 5 * time.Minute,
	GetKey: func(item *JobData) string {
		return item.SId
	},
	Disabled: !config.Feature.HasVault(),
})

func QueueJob(sid string) {
	_ = queue.Queue(JobData{SId: sid})
}
