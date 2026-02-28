package torrent_stream

import (
	"context"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/job"
	"github.com/MunifTanjim/stremthru/internal/job/job_queue"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
)

type MediaInfoProbeJobData struct {
	Hash string
	Path string
	Link string
}

var mediaInfoProbeQueue = job_queue.NewMemoryJobQueue(job_queue.JobQueueConfig[MediaInfoProbeJobData]{
	GetKey: func(item *MediaInfoProbeJobData) string {
		return item.Hash + ":" + item.Path
	},
	DebounceTime: 5 * time.Minute,
	Disabled:     !config.Feature.IsEnabled(config.FeatureProbeMediaInfo),
})

func QueueMediaInfoProbe(hash, path, link string) {
	mediaInfoProbeQueue.Queue(MediaInfoProbeJobData{
		Hash: hash,
		Path: path,
		Link: link,
	})
}

var _ = job.NewScheduler(&job.SchedulerConfig[MediaInfoProbeJobData]{
	Id:       "probe-media-info",
	Title:    "Probe Media Info",
	Interval: 10 * time.Minute,
	Queue:    mediaInfoProbeQueue,
	Disabled: !config.Feature.IsEnabled(config.FeatureProbeMediaInfo),
	ShouldSkip: func() bool {
		return mediaInfoProbeQueue.IsEmpty()
	},
	RunExclusive: true,
	Executor: func(j *job.Scheduler[MediaInfoProbeJobData]) error {
		log := j.Logger()

		j.JobQueue().Process(func(data MediaInfoProbeJobData) error {
			if existing := HasMediaInfo(data.Hash, data.Path); existing {
				log.Trace("media info already exists", "hash", data.Hash, "path", data.Path)
				return nil
			}

			mi, err := media_info.Probe(context.Background(), data.Link)
			if err != nil {
				log.Error("failed to probe media info", "hash", data.Hash, "path", data.Path, "error", err)
				return nil
			}

			if err := SetMediaInfo(data.Hash, data.Path, mi); err != nil {
				log.Error("failed to save media info", "error", err, "hash", data.Hash, "path", data.Path)
				return nil
			}

			log.Info("saved media info", "hash", data.Hash, "path", data.Path)
			return nil
		})
		return nil
	},
})
