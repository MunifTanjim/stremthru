package util

import (
	"bufio"
	"errors"
	"iter"
	"os"
	"slices"
	"strings"
)

type SimpleTSVDataset[T any] struct {
	*Dataset
	get_row_key      func(row []string) string
	has_headers      bool
	is_valid_headers func(headers []string) bool
	no_diff          bool
	parse_row        func(row []string) (*T, error)
	column_count     int
	current_line     int64
	w                *DatasetWriter[T]
}

type SimpleTSVDatasetConfig[T any] struct {
	DatasetConfig
	GetRowKey      func(row []string) string
	HasHeaders     bool
	IsValidHeaders func(headers []string) bool
	NoDiff         bool
	ParseRow       func(row []string) (*T, error)
	Writer         *DatasetWriter[T]
}

func NewSimpleTSVDataset[T any](conf *SimpleTSVDatasetConfig[T]) *SimpleTSVDataset[T] {
	ds := SimpleTSVDataset[T]{
		Dataset:          NewDataset((*DatasetConfig)(&conf.DatasetConfig)),
		get_row_key:      conf.GetRowKey,
		has_headers:      conf.HasHeaders,
		is_valid_headers: conf.IsValidHeaders,
		no_diff:          conf.NoDiff,
		parse_row:        conf.ParseRow,
		w:                conf.Writer,
	}
	return &ds
}

func (ds *SimpleTSVDataset[T]) newReader(file *os.File) *bufio.Scanner {
	r := bufio.NewScanner(file)
	if ds.has_headers {
		headers := ds.nextRow(r)
		if !ds.is_valid_headers(headers) {
			ds.log.Error("invalid headers", "headers", headers)
			return nil
		}
		ds.column_count = len(headers)
	}
	return r
}

func (ds *SimpleTSVDataset[T]) nextRow(r *bufio.Scanner) []string {
	for {
		if r.Scan() {
			ds.current_line++
			line := r.Text()
			record := strings.Split(line, "\t")
			if ds.column_count == 0 {
				ds.column_count = len(record)
			}
			if len(record) != ds.column_count {
				ds.log.Debug("wrong number of fields", "line", ds.current_line)
				continue
			}
			return record
		} else {
			err := r.Err()
			if err != nil {
				ds.log.Trace("failed to read row", "error", err)
			}
			return nil
		}
	}
}

func (ds SimpleTSVDataset[T]) allRows(r *bufio.Scanner) iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		for {
			if row := ds.nextRow(r); row == nil || !yield(row) {
				return
			}
		}
	}
}

func (ds SimpleTSVDataset[T]) diffRows(oldR, newR *bufio.Scanner) iter.Seq[[]string] {
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

func (ds *SimpleTSVDataset[T]) processAll() error {
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

func (ds *SimpleTSVDataset[T]) processDiff() error {
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
		if err := os.RemoveAll(lastFilePath); err != nil {
			ds.log.Error("failed to remove last file with invalid headers", "error", err)
			return errors.New("failed to create reader for last file")
		}
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

func (ds *SimpleTSVDataset[T]) Process() error {
	if err := ds.Init(); err != nil {
		return err
	}

	if ds.no_diff || ds.prev_filename == "" || ds.prev_filename == ds.curr_filename {
		return ds.processAll()
	}
	return ds.processDiff()
}
