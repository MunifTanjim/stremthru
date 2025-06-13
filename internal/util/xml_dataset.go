package util

import (
	"encoding/xml"
	"errors"
	"io"
	"iter"
	"log/slog"
	"os"
)

type XMLDataset[T any, I any] struct {
	*Dataset
	list_tag_name string
	item_tag_name string
	no_diff       bool
	prepare       func(item *T) *I
	get_item_key  func(item *I) string
	is_item_equal func(a *I, b *I) bool
	w             *DatasetWriter[I]
}

type XMLDatasetConfig[T any, I any] struct {
	DatasetConfig
	ListTagName string
	ItemTagName string
	NoDiff      bool
	Prepare     func(item *T) *I
	GetItemKey  func(item *I) string
	IsItemEqual func(a *I, b *I) bool
	Writer      *DatasetWriter[I]
}

func NewXMLDataset[T any, I any](conf *XMLDatasetConfig[T, I]) *XMLDataset[T, I] {
	ds := XMLDataset[T, I]{
		Dataset:       NewDataset((*DatasetConfig)(&conf.DatasetConfig)),
		list_tag_name: conf.ListTagName,
		item_tag_name: conf.ItemTagName,
		no_diff:       conf.NoDiff,
		get_item_key:  conf.GetItemKey,
		is_item_equal: conf.IsItemEqual,
		prepare:       conf.Prepare,
		w:             conf.Writer,
	}
	return &ds
}

type XMLDatasetReader[T any, I any] struct {
	decoder       *xml.Decoder
	get_item_key  func(item *I) string
	inside_list   bool
	is_done       bool
	item_tag_name string
	list_tag_name string
	log           *slog.Logger
	prepare       func(item *T) *I
}

type XMLDatasetReaderConfig[T any, I any] struct {
	File        io.Reader
	GetItemKey  func(item *I) string
	ItemTagName string
	ListTagName string
	Log         *slog.Logger
	Prepare     func(item *T) *I
}

func NewXMLDatasetReader[T any, I any](conf *XMLDatasetReaderConfig[T, I]) *XMLDatasetReader[T, I] {
	if conf.Prepare == nil {
		conf.Prepare = func(item *T) *I {
			return any(item).(*I)
		}
	}
	return &XMLDatasetReader[T, I]{
		decoder:       xml.NewDecoder(conf.File),
		get_item_key:  conf.GetItemKey,
		inside_list:   false,
		is_done:       false,
		item_tag_name: conf.ItemTagName,
		list_tag_name: conf.ListTagName,
		log:           conf.Log,
		prepare:       conf.Prepare,
	}
}

func (ds XMLDataset[T, I]) NewReader(file *os.File) *XMLDatasetReader[T, I] {
	return NewXMLDatasetReader(&XMLDatasetReaderConfig[T, I]{
		File:        file,
		GetItemKey:  ds.get_item_key,
		ItemTagName: ds.item_tag_name,
		ListTagName: ds.list_tag_name,
		Log:         ds.log,
		Prepare:     ds.prepare,
	})
}

func (r *XMLDatasetReader[T, I]) NextItem() *I {
	if r.is_done {
		return nil
	}

	for {
		tok, err := r.decoder.Token()
		if err != nil {
			if err == io.EOF {
				r.is_done = true
			} else {
				r.log.Debug("failed to read token", "error", err)
			}
			break
		}

		switch elem := tok.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case r.list_tag_name:
				r.inside_list = true
			case r.item_tag_name:
				if r.inside_list {
					var raw_item T
					if err := r.decoder.DecodeElement(&raw_item, &elem); err != nil {
						r.log.Debug("failed to decode item", "error", err)
						continue
					}
					item := r.prepare(&raw_item)
					if item == nil {
						r.log.Debug("prepared item is nil, skipping", "item", raw_item)
						continue
					}
					if r.get_item_key(item) == "" {
						r.log.Debug("item key is missing, skipping", "item", raw_item)
						continue
					}
					return item
				}
			}
		case xml.EndElement:
			if elem.Name.Local == r.list_tag_name {
				r.inside_list = false
				r.is_done = true
				return nil
			}
		}
	}

	return nil
}

func (ds XMLDataset[T, I]) allItems(r *XMLDatasetReader[T, I]) iter.Seq[*I] {
	return func(yield func(*I) bool) {
		for {
			if item := r.NextItem(); item == nil || !yield(item) {
				return
			}
		}
	}
}

func (ds XMLDataset[T, I]) diffItems(oldR, newR *XMLDatasetReader[T, I]) iter.Seq[*I] {
	return func(yield func(*I) bool) {
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

func (ds XMLDataset[T, I]) processAll() error {
	ds.log.Info("processing whole dataset...")

	filePath := ds.filePath(ds.curr_filename)
	file, err := os.Open(filePath)
	if err != nil {
		return aError{"failed to open file", err}
	}
	defer file.Close()

	r := ds.NewReader(file)
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

func (ds XMLDataset[T, I]) processDiff() error {
	ds.log.Info("processing diff dataset...")

	lastFilePath := ds.filePath(ds.prev_filename)
	lastFile, err := os.Open(lastFilePath)
	if err != nil {
		return aError{"failed to open last file", err}
	}
	defer lastFile.Close()

	newFilePath := ds.filePath(ds.curr_filename)
	newFile, err := os.Open(newFilePath)
	if err != nil {
		return aError{"failed to open new file", err}
	}
	defer newFile.Close()

	lastR := ds.NewReader(lastFile)
	if lastR == nil {
		return errors.New("failed to create reader for last file")
	}
	newR := ds.NewReader(newFile)
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

func (ds XMLDataset[T, I]) Process() error {
	if err := ds.Init(); err != nil {
		return err
	}
	if ds.no_diff || ds.prev_filename == "" || ds.prev_filename == ds.curr_filename {
		return ds.processAll()
	}
	return ds.processDiff()
}
