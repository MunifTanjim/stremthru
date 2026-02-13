package nzb_info

import (
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
)

const JobQueueName = "nzb"

type JobData struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Category string `json:"category"`
	Password string `json:"password"`
	User     string `json:"user"`
	Priority int    `json:"priority"`
}

var queue = job_queue.NewPersistentJobQueue(JobQueueName, job_queue.JobQueueConfig[JobData]{
	GetKey: func(item *JobData) string {
		return HashNZBFileLink(item.URL)
	},
})

type JobEntry = job_queue.JobQueueEntry[JobData]

func QueueJob(user, name, url, category string, priority int, password string) (string, error) {
	err := scheduler.Trigger(JobData{
		Name:     name,
		URL:      url,
		Category: category,
		Password: password,
		User:     user,
		Priority: priority,
	})
	if err != nil {
		return "", err
	}
	return HashNZBFileLink(url), nil
}

func GetAllJob() ([]JobEntry, error) {
	return job_queue.GetEntriesByName[JobData](JobQueueName)
}

func GetJobById(id string) (*JobEntry, error) {
	return job_queue.GetEntryByKey[JobData](JobQueueName, id)
}

func DeleteJob(id string) error {
	return job_queue.DeleteEntries(JobQueueName, []string{id})
}
