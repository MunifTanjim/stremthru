package job_queue

import (
	"errors"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
)

type PersistentJobQueue[T any] struct {
	name         string
	getKey       func(item *T) string
	getGroupKey  func(item *T) string
	debounceTime time.Duration
	backoffDelay time.Duration
	maxRetry     int
	disabled     bool
}

func exponentialBackoff(errorCount int, delay time.Duration) time.Duration {
	return time.Duration(1<<max(errorCount-1, 0)) * delay
}

func (q *PersistentJobQueue[T]) Queue(item T, priority ...int) error {
	if q.disabled {
		return nil
	}
	key := q.getKey(&item)
	processAfter := time.Now().Add(q.debounceTime)
	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}
	if err := QueueEntry(q.name, item, key, processAfter, p); err != nil {
		log.Error("JobQueue persist failed", "error", err, "name", q.name)
		return err
	}
	return nil
}

func (q *PersistentJobQueue[T]) IsEmpty() bool {
	exists, err := EntryExists(q.name)
	if err != nil {
		log.Error("JobQueue isEmpty check failed", "error", err, "name", q.name)
		return true
	}
	return !exists
}

func (q *PersistentJobQueue[T]) Process(f func(item T) error) {
	for {
		entry, err := GetFirstEntry[T](q.name)
		if err != nil {
			log.Error("JobQueue dequeue failed", "error", err, "name", q.name)
			return
		}
		if entry == nil {
			return
		}
		if err := SetEntriesProcessing(q.name, []string{entry.Key}); err != nil {
			log.Error("JobQueue set processing failed", "error", err, "name", q.name, "key", entry.Key)
			return
		}
		if err := f(entry.Payload.Data); err != nil {
			var delayed *ErrJobQueueItemDelayed
			if errors.As(err, &delayed) {
				processAfter := time.Now().Add(delayed.RetryAfter)
				if err := DelayEntries(q.name, []string{entry.Key}, processAfter); err != nil {
					log.Error("JobQueue delay failed", "error", err, "name", q.name, "key", entry.Key)
				}
				log.Debug("JobQueue process delayed", "name", q.name, "key", entry.Key, "retry_after", delayed.RetryAfter)
				continue
			}
			errs := append(entry.Error, err.Error())
			if len(errs) > q.maxRetry {
				if err := SetEntryDead(q.name, entry.Key, errs); err != nil {
					log.Error("JobQueue set dead failed", "error", err, "name", q.name, "key", entry.Key)
				}
				log.Error("JobQueue process dead", "error", err, "name", q.name, "key", entry.Key)
			} else {
				processAfter := time.Now().Add(exponentialBackoff(len(errs), q.backoffDelay))
				if err := SetEntryFailed(q.name, entry.Key, errs, processAfter); err != nil {
					log.Error("JobQueue set failed failed", "error", err, "name", q.name, "key", entry.Key)
				}
				log.Error("JobQueue process failed", "error", err, "name", q.name, "key", entry.Key)
			}
			continue
		}
		if err := SetEntriesDone(q.name, []string{entry.Key}); err != nil {
			log.Error("JobQueue set done failed", "error", err, "name", q.name, "key", entry.Key)
			return
		}
	}
}

func (q *PersistentJobQueue[T]) ProcessGroup(f func(groupKey string, items []T) error) {
	type entryItem struct {
		key    string
		item   T
		errors db.JSONStringList
	}
	for {
		entries, err := GetAllPendingEntries[T](q.name)
		if err != nil {
			log.Error("JobQueue dequeue failed", "error", err, "name", q.name)
			return
		}
		if len(entries) == 0 {
			return
		}
		allKeys := make([]string, len(entries))
		for i, entry := range entries {
			allKeys[i] = entry.Key
		}
		if err := SetEntriesProcessing(q.name, allKeys); err != nil {
			log.Error("JobQueue set processing failed", "error", err, "name", q.name)
			return
		}
		byGroupKey := map[string][]entryItem{}
		for _, entry := range entries {
			groupKey := q.getGroupKey(&entry.Payload.Data)
			byGroupKey[groupKey] = append(byGroupKey[groupKey], entryItem{key: entry.Key, item: entry.Payload.Data, errors: entry.Error})
		}
		for groupKey, entries := range byGroupKey {
			items := make([]T, len(entries))
			for i, ei := range entries {
				items[i] = ei.item
			}
			if err := f(groupKey, items); err != nil {
				var delayed *ErrJobQueueItemDelayed
				if errors.As(err, &delayed) {
					processAfter := time.Now().Add(delayed.RetryAfter)
					keys := make([]string, len(entries))
					for i, ei := range entries {
						keys[i] = ei.key
					}
					if err := DelayEntries(q.name, keys, processAfter); err != nil {
						log.Error("JobQueue delay failed", "error", err, "name", q.name, "group_key", groupKey)
					}
					log.Debug("JobQueue processGroup delayed", "group_key", groupKey, "retry_after", delayed.RetryAfter)
				} else {
					for _, ei := range entries {
						errs := append(ei.errors, err.Error())
						if len(errs) > q.maxRetry {
							if err := SetEntryDead(q.name, ei.key, errs); err != nil {
								log.Error("JobQueue set dead failed", "error", err, "name", q.name, "key", ei.key, "group_key", groupKey)
							}
							log.Error("JobQueue processGroup dead", "name", q.name, "key", ei.key, "group_key", groupKey, "errors", errs)
						} else {
							processAfter := time.Now().Add(exponentialBackoff(len(errs), q.backoffDelay))
							if err := SetEntryFailed(q.name, ei.key, errs, processAfter); err != nil {
								log.Error("JobQueue set failed failed", "error", err, "name", q.name, "key", ei.key, "group_key", groupKey)
							}
							log.Error("JobQueue processGroup failed", "error", errs[len(errs)-1], "name", q.name, "key", ei.key, "group_key", groupKey)
						}
					}
				}
				continue
			}
			keys := make([]string, len(entries))
			for i, entry := range entries {
				keys[i] = entry.key
			}
			if err := SetEntriesDone(q.name, keys); err != nil {
				log.Error("JobQueue set done failed", "error", err, "name", q.name, "group_key", groupKey)
				return
			}
		}
	}
}
