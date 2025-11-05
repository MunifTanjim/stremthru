package util

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger/log"
)

type DatasetWriter[T any] struct {
	batch_idx      int
	batch_size     int
	idx            int
	is_done        bool
	log            *log.Logger
	items          []T
	upsert         func([]T) error
	sleep_duration time.Duration
}

type DatasetWriterConfig[T any] struct {
	BatchSize     int
	Log           *log.Logger
	Upsert        func([]T) error
	SleepDuration time.Duration
}

func NewDatasetWriter[T any](conf DatasetWriterConfig[T]) *DatasetWriter[T] {
	if conf.BatchSize == 0 {
		conf.BatchSize = 500
	}
	if conf.SleepDuration == 0 {
		conf.SleepDuration = 250 * time.Millisecond
	}
	dsw := DatasetWriter[T]{
		batch_idx:      0,
		batch_size:     conf.BatchSize,
		idx:            0,
		is_done:        false,
		log:            conf.Log,
		items:          make([]T, conf.BatchSize),
		upsert:         conf.Upsert,
		sleep_duration: conf.SleepDuration,
	}
	return &dsw
}

func (w *DatasetWriter[T]) Write(t *T) error {
	if w.is_done {
		return nil
	}

	if t == nil {
		return nil
	}

	w.items[w.idx] = *t
	w.idx++
	if w.idx == w.batch_size {
		w.batch_idx++
		if err := w.upsert(w.items); err != nil {
			return err
		}
		w.log.Info("upserted items", "count", w.batch_idx*w.batch_size)
		w.idx = 0
		time.Sleep(w.sleep_duration)
	}
	return nil
}

func (w *DatasetWriter[T]) Done() error {
	if w.is_done {
		return nil
	}

	w.is_done = true

	w.items = w.items[0:w.idx]
	if err := w.upsert(w.items); err != nil {
		return err
	}
	w.is_done = true
	w.log.Info("upserted items", "count", w.batch_idx*w.batch_size+w.idx)
	return nil
}
