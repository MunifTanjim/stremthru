package util

import (
	"encoding/xml"
	"errors"
	"io"
	"iter"
	"os"
)

type XMLDataset[T any, I any] struct {
	*Dataset
	list_tag_name string
	item_tag_name string
	prepare       func(item *T) *I
	get_item_key  func(item *I) string
	is_item_equal func(a *I, b *I) bool
	w             *DatasetWriter[I]
}

type XMLDatasetConfig[T any, I any] struct {
	DatasetConfig
	ListTagName string
	ItemTagName string
	Prepare     func(item *T) *I
	GetItemKey  func(item *I) string
	IsItemEqual func(a *I, b *I) bool
	Writer      *DatasetWriter[I]
}

func NewXMLDataset[T any, I any](conf *XMLDatasetConfig[T, I]) *XMLDataset[T, I] {
	if conf.Prepare == nil {
		conf.Prepare = func(item *T) *I {
			return any(item).(*I)
		}
	}
	ds := XMLDataset[T, I]{
		Dataset:       NewDataset((*DatasetConfig)(&conf.DatasetConfig)),
		list_tag_name: conf.ListTagName,
		item_tag_name: conf.ItemTagName,
		get_item_key:  conf.GetItemKey,
		is_item_equal: conf.IsItemEqual,
		prepare:       conf.Prepare,
		w:             conf.Writer,
	}
	return &ds
}

type XMLDatasetReader struct {
	decoder     *xml.Decoder
	inside_list bool
	is_done     bool
}

func (ds XMLDataset[T, I]) newReader(file *os.File) *XMLDatasetReader {
	return &XMLDatasetReader{
		decoder:     xml.NewDecoder(file),
		inside_list: false,
		is_done:     false,
	}
}

func (ds XMLDataset[T, I]) nextItem(r *XMLDatasetReader) *I {
	if r.is_done {
		return nil
	}

	for {
		tok, err := r.decoder.Token()
		if err != nil {
			if err == io.EOF {
				r.is_done = true
			} else {
				ds.log.Debug("failed to read token", "error", err)
			}
			break
		}

		switch elem := tok.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case ds.list_tag_name:
				r.inside_list = true
			case ds.item_tag_name:
				if r.inside_list {
					var raw_item T
					if err := r.decoder.DecodeElement(&raw_item, &elem); err != nil {
						ds.log.Debug("failed to decode item", "error", err)
						continue
					}
					item := ds.prepare(&raw_item)
					if item == nil {
						ds.log.Debug("prepared item is nil, skipping", "item", raw_item)
						continue
					}
					if ds.get_item_key(item) == "" {
						ds.log.Debug("item key is missing, skipping", "item", raw_item)
						continue
					}
					return item
				}
			}
		case xml.EndElement:
			if elem.Name.Local == ds.list_tag_name {
				r.inside_list = false
				r.is_done = true
				return nil
			}
		}
	}

	return nil
}

func (ds XMLDataset[T, I]) allItems(r *XMLDatasetReader) iter.Seq[*I] {
	return func(yield func(*I) bool) {
		for {
			if item := ds.nextItem(r); item == nil || !yield(item) {
				return
			}
		}
	}
}

func (ds XMLDataset[T, I]) diffItems(oldR, newR *XMLDatasetReader) iter.Seq[*I] {
	return func(yield func(*I) bool) {
		oldItem := ds.nextItem(oldR)
		newItem := ds.nextItem(newR)

		for oldItem != nil && newItem != nil {
			oldKey := ds.get_item_key(oldItem)
			newKey := ds.get_item_key(newItem)

			switch {
			case oldKey < newKey:
				// removed
				oldItem = ds.nextItem(oldR)
			case oldKey > newKey:
				// added
				if !yield(newItem) {
					return
				}
				newItem = ds.nextItem(newR)
			default:
				if !ds.is_item_equal(oldItem, newItem) {
					// changed
					if !yield(newItem) {
						return
					}
				}
				oldItem = ds.nextItem(oldR)
				newItem = ds.nextItem(newR)
			}
		}

		for newItem != nil {
			if !yield(newItem) {
				return
			}
			newItem = ds.nextItem(newR)
		}

		for oldItem != nil {
			// removed
			oldItem = ds.nextItem(oldR)
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

	r := ds.newReader(file)
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

	lastR := ds.newReader(lastFile)
	if lastR == nil {
		return errors.New("failed to create reader for last file")
	}
	newR := ds.newReader(newFile)
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
	if ds.prev_filename == "" || ds.prev_filename == ds.curr_filename {
		return ds.processAll()
	}
	return ds.processDiff()
}
