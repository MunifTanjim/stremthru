package nzb_info

import (
	"context"

	"github.com/MunifTanjim/stremthru/internal/job"
	"github.com/MunifTanjim/stremthru/internal/logger"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

const schedulerId = "process-nzb"

var log = logger.Scoped("job/" + schedulerId)

var scheduler = job.NewScheduler(&job.SchedulerConfig[JobData]{
	Id:           schedulerId,
	Title:        "Process NZB",
	RunExclusive: true,
	Queue:        queue,
	Executor: func(j *job.Scheduler[JobData]) error {
		j.JobQueue().Process(func(data JobData) error {
			nzbFile, err := fetchNZBFile(data.URL, data.Name, log, nil)
			if err != nil {
				return err
			}

			nzbDoc, err := nzb.ParseBytes(nzbFile.Blob)
			if err != nil {
				return err
			}

			hash := HashNZBFileLink(data.URL)

			name := data.Name
			if name == "" {
				name = nzbDoc.GetMeta("title")
			}
			if name == "" {
				name = nzbFile.Name
			}

			password := nzbDoc.GetMeta("password")
			if password == "" {
				password = data.Password
			}

			info := &NZBInfo{
				Hash:      hash,
				Name:      name,
				Size:      nzbDoc.TotalSize(),
				FileCount: nzbDoc.FileCount(),
				Password:  password,
				URL:       data.URL,
				User:      data.User,
			}

			pool, err := usenetmanager.GetPool()
			if err != nil {
				return err
			}
			content, err := pool.InspectNZBContent(context.Background(), nzbDoc, password)
			if err != nil {
				log.Warn("failed to inspect nzb content", "error", err)
				return err
			}
			info.ContentFiles.Data = content.Files
			info.Streamable = content.Streamable

			return Upsert(info)
		})
		return nil
	},
	ShouldSkip: func() bool {
		pool, err := usenetmanager.GetPool()
		return err != nil || pool.CountProviders() == 0
	},
})
