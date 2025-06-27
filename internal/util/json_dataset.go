package util

import (
	"errors"
	"iter"
	"log/slog"
	"os"
)

type JSONDataset[T any] struct {
	*Dataset
	get_item_key  func(item *T) string
	get_seq       func(blob []byte) (iter.Seq[*T], error)
	is_item_equal func(a *T, b *T) bool
	w             *DatasetWriter[T]
}

type JSONDatasetConfig[T any] struct {
	DatasetConfig
	ListTagName string
	ItemTagName string
	NoDiff      bool
	GetItemKey  func(item *T) string
	GetSeq      func(blob []byte) (iter.Seq[*T], error)
	IsItemEqual func(a *T, b *T) bool
	Writer      *DatasetWriter[T]
}

func NewJSONDataset[T any](conf *JSONDatasetConfig[T]) *JSONDataset[T] {
	ds := JSONDataset[T]{
		Dataset:       NewDataset((*DatasetConfig)(&conf.DatasetConfig)),
		get_item_key:  conf.GetItemKey,
		get_seq:       conf.GetSeq,
		is_item_equal: conf.IsItemEqual,
		w:             conf.Writer,
	}
	return &ds
}

type JSONDatasetReader[T any] struct {
	get_item_key func(item *T) string
	is_done      bool
	log          *slog.Logger
	seq_next     func() (*T, bool)
	seq_stop     func()
}

type JSONDatasetReaderConfig[T any] struct {
	Blob       []byte
	GetItemKey func(item *T) string
	GetSeq     func(blob []byte) (iter.Seq[*T], error)
	Log        *slog.Logger
}

func NewJSONDatasetReader[T any](conf *JSONDatasetReaderConfig[T]) *JSONDatasetReader[T] {
	seq, err := conf.GetSeq(conf.Blob)
	if err != nil {
		conf.Log.Error("failed to get sequence from blob", "error", err)
		return nil
	}
	seq_next, seq_stop := iter.Pull(seq)
	return &JSONDatasetReader[T]{
		get_item_key: conf.GetItemKey,
		is_done:      false,
		log:          conf.Log,
		seq_next:     seq_next,
		seq_stop:     seq_stop,
	}
}

func (ds JSONDataset[T]) NewReader(blob []byte) *JSONDatasetReader[T] {
	return NewJSONDatasetReader(&JSONDatasetReaderConfig[T]{
		Blob:       blob,
		GetItemKey: ds.get_item_key,
		GetSeq:     ds.get_seq,
		Log:        ds.log,
	})
}

func (r *JSONDatasetReader[T]) NextItem() *T {
	if r.is_done {
		return nil
	}

	for {
		item, ok := r.seq_next()
		if !ok {
			r.is_done = true
			return nil
		}
		if r.get_item_key(item) == "" {
			r.log.Debug("item key is missing, skipping", "item", item)
			continue
		}
		return item
	}
}

func (ds JSONDataset[T]) allItems(r *JSONDatasetReader[T]) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		defer r.seq_stop()
		for {
			if item := r.NextItem(); item == nil || !yield(item) {
				return
			}
		}
	}
}

func (ds JSONDataset[T]) diffItems(oldR, newR *JSONDatasetReader[T]) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		defer oldR.seq_stop()
		defer newR.seq_stop()

		oldItem := oldR.NextItem()
		newItem := newR.NextItem()

		for oldItem != nil && newItem != nil {
			oldKey := oldR.get_item_key(oldItem)
			newKey := newR.get_item_key(newItem)

			switch {
			case oldKey < newKey:
				// removed
				oldItem = oldR.NextItem()
			case oldKey > newKey:
				// added
				if !yield(newItem) {
					return
				}
				newItem = newR.NextItem()
			default:
				if !ds.is_item_equal(oldItem, newItem) {
					// changed
					if !yield(newItem) {
						return
					}
				}
				oldItem = oldR.NextItem()
				newItem = newR.NextItem()
			}
		}

		for newItem != nil {
			if !yield(newItem) {
				return
			}
			newItem = newR.NextItem()
		}

		for oldItem != nil {
			// removed
			oldItem = oldR.NextItem()
		}
	}
}

func (ds JSONDataset[T]) processAll() error {
	ds.log.Info("processing whole dataset...")

	filePath := ds.filePath(ds.curr_filename)
	blob, err := os.ReadFile(filePath)
	if err != nil {
		return aError{"failed to open file", err}
	}

	r := ds.NewReader(blob)
	if r == nil {
		return errors.New("failed to create reader")
	}

	for item := range ds.allItems(r) {
		if err := ds.w.Write(item); err != nil {
			return err
		}
	}

	return ds.w.Done()
}

func (ds JSONDataset[T]) processDiff() error {
	ds.log.Info("processing diff dataset...")

	lastFilePath := ds.filePath(ds.prev_filename)
	lastBlob, err := os.ReadFile(lastFilePath)
	if err != nil {
		return aError{"failed to open last file", err}
	}

	newFilePath := ds.filePath(ds.curr_filename)
	newBlob, err := os.ReadFile(newFilePath)
	if err != nil {
		return aError{"failed to open new file", err}
	}

	lastR := ds.NewReader(lastBlob)
	if lastR == nil {
		return errors.New("failed to create reader for last file")
	}
	newR := ds.NewReader(newBlob)
	if newR == nil {
		return errors.New("failed to create reader for new file")
	}

	for item := range ds.diffItems(lastR, newR) {
		err = ds.w.Write(item)
		if err != nil {
			return err
		}
	}

	return ds.w.Done()
}

func (ds JSONDataset[T]) Process() error {
	if err := ds.Init(); err != nil {
		return err
	}
	if ds.prev_filename == "" || ds.prev_filename == ds.curr_filename {
		return ds.processAll()
	}
	return ds.processDiff()
}
