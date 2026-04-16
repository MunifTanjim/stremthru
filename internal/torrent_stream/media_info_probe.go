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

	Link string // for ffprobe

	StoreCode  string // for store probing
	StoreToken string
	LinkId     string
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

func QueueStoreMediaInfoProbe(hash, path, storeCode, storeToken, linkId string) {
	mediaInfoProbeQueue.Queue(MediaInfoProbeJobData{
		Hash:       hash,
		Path:       path,
		StoreCode:  storeCode,
		StoreToken: storeToken,
		LinkId:     linkId,
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
			existing := GetMediaInfo(data.Hash, data.Path)

			candidate := &media_info.MediaInfo{Source: data.StoreCode}
			if !candidate.ShouldOverwrite(existing) {
				log.Trace("media info already exists, skipping probe", "hash", data.Hash, "path", data.Path)
				return nil
			}

			var mi *media_info.MediaInfo
			var err error

			if data.StoreCode != "" {
				mi, err = media_info.ProbeStore(data.StoreCode, data.StoreToken, data.LinkId)
			} else {
				mi, err = media_info.Probe(context.Background(), data.Link)
			}

			if err != nil {
				log.Error("failed to probe media info", "hash", data.Hash, "path", data.Path, "error", err)
				return nil
			}

			if err := SetMediaInfo(data.Hash, data.Path, mi); err != nil {
				log.Error("failed to save media info", "error", err, "hash", data.Hash, "path", data.Path)
				return nil
			}

			log.Info("saved media info", "hash", data.Hash, "path", data.Path, "src", mi.Source)
			return nil
		})
		return nil
	},
})
