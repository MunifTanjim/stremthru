package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type LetterboxdListSyncerQueueItem struct {
	ListId string
}

var LetterboxdListSyncerQueue = WorkerQueue[LetterboxdListSyncerQueueItem]{
	debounceTime: func() time.Duration {
		if config.Integration.Letterboxd.IsEnabled() {
			return 1 * time.Minute
		}
		return 5 * time.Minute
	}(),
	getKey: func(item LetterboxdListSyncerQueueItem) string {
		return item.ListId
	},
	transform: func(item *LetterboxdListSyncerQueueItem) *LetterboxdListSyncerQueueItem {
		return item
	},
	Disabled: !config.Feature.HasStremioList() || (!config.Integration.Letterboxd.IsEnabled() && !config.Integration.Letterboxd.IsPiggybacked()),
}
