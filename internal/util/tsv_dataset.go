package util

import (
	"encoding/csv"
	"errors"
	"io"
	"iter"
	"os"
	"slices"
)

type TSVDataset[T any] struct {
	*Dataset
	get_row_key      func(row []string) string
	has_headers      bool
	is_valid_headers func(headers []string) bool
	parse_row        func(row []string) (*T, error)
	w                *DatasetWriter[T]
}

type TSVDatasetConfig[T any] struct {
	DatasetConfig
	GetRowKey      func(row []string) string
	HasHeaders     bool
	IsValidHeaders func(headers []string) bool
	ParseRow       func(row []string) (*T, error)
	Writer         *DatasetWriter[T]
}

func NewTSVDataset[T any](conf *TSVDatasetConfig[T]) *TSVDataset[T] {
	ds := TSVDataset[T]{
		Dataset:          NewDataset((*DatasetConfig)(&conf.DatasetConfig)),
		get_row_key:      conf.GetRowKey,
		has_headers:      conf.HasHeaders,
		is_valid_headers: conf.IsValidHeaders,
		parse_row:        conf.ParseRow,
		w:                conf.Writer,
	}
	return &ds
}

func (ds TSVDataset[T]) newReader(file *os.File) *csv.Reader {
	r := csv.NewReader(file)
	r.Comma = '\t'
	r.LazyQuotes = true
	r.ReuseRecord = true
	if ds.has_headers {
		headers := ds.nextRow(r)
		if !ds.is_valid_headers(headers) {
			ds.log.Error("invalid headers", "headers", headers)
			return nil
		}
	}
	return r
}

func (ds TSVDataset[T]) nextRow(r *csv.Reader) []string {
	for {
		if record, err := r.Read(); err == io.EOF || err == nil {
			if err == io.EOF {
				return nil
			}
			return record
		} else {
			ds.log.Debug("failed to read row", "error", err)
		}
	}
}

func (ds TSVDataset[T]) allRows(r *csv.Reader) iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		for {
			if row := ds.nextRow(r); row == nil || !yield(row) {
				return
			}
		}
	}
}

func (ds TSVDataset[T]) diffRows(oldR, newR *csv.Reader) iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		oldRec := ds.nextRow(oldR)
		newRec := ds.nextRow(newR)

		for oldRec != nil && newRec != nil {
			oldKey := ds.get_row_key(oldRec)
			newKey := ds.get_row_key(newRec)

			switch {
			case oldKey < newKey:
				// removed
				oldRec = ds.nextRow(oldR)
			case oldKey > newKey:
				// added
				if !yield(newRec) {
					return
				}
				newRec = ds.nextRow(newR)
			default:
				if !slices.Equal(oldRec, newRec) {
					// changed
					if !yield(newRec) {
						return
					}
				}
				oldRec = ds.nextRow(oldR)
				newRec = ds.nextRow(newR)
			}
		}

		for newRec != nil {
			if !yield(newRec) {
				return
			}
			newRec = ds.nextRow(newR)
		}

		for oldRec != nil {
			// removed
			oldRec = ds.nextRow(oldR)
		}
	}
}

func (ds *TSVDataset[T]) processAll() error {
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

	for row := range ds.allRows(r) {
		t, err := ds.parse_row(row)
		if err != nil {
			return err
		}

		if err := ds.w.Write(t); err != nil {
			return err
		}
	}

	return ds.w.Done()
}

func (ds *TSVDataset[T]) processDiff() error {
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

	for row := range ds.diffRows(lastR, newR) {
		item, err := ds.parse_row(row)
		if err != nil {
			return err
		}
		err = ds.w.Write(item)
		if err != nil {
			return err
		}
	}

	return ds.w.Done()
}

func (ds *TSVDataset[T]) Process() error {
	if err := ds.Init(); err != nil {
		return err
	}

	if ds.prev_filename == "" || ds.prev_filename == ds.curr_filename {
		return ds.processAll()
	}
	return ds.processDiff()
}
