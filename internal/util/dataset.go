package util

import (
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type aError struct {
	string
	cause error
}

func (e aError) Error() string {
	return e.string + "\n" + e.cause.Error()
}

type Dataset struct {
	Name                   string
	archive_ext            string
	curr_filename          string
	download_dir           string
	download_dir_exists    bool
	download_headers       map[string]string
	get_download_file_time func() time.Time
	is_stale               func(time.Time) bool
	log                    *slog.Logger
	prefix_time_format     string
	prev_filename          string
	url                    string
}

type DatasetConfig struct {
	Archive             string
	DownloadDir         string
	DownloadHeaders     map[string]string
	GetDownloadFileTime func() time.Time
	IsStale             func(time.Time) bool
	Log                 *slog.Logger
	Name                string
	URL                 string
}

func NewDataset(conf *DatasetConfig) *Dataset {
	if conf.Name == "" {
		conf.Name = filepath.Base(conf.URL)
	}
	if conf.DownloadDir == "" {
		panic("DownloadDir must be set")
	}
	if conf.GetDownloadFileTime == nil {
		conf.GetDownloadFileTime = func() time.Time {
			return time.Now()
		}
	}

	if conf.Archive != "" && conf.Archive != "gz" {
		panic("Unsupported archive format: " + conf.Archive)
	}
	archive_ext := ""
	if conf.Archive != "" {
		archive_ext = "." + conf.Archive
		if strings.HasSuffix(conf.Name, archive_ext) {
			conf.Name = strings.TrimSuffix(conf.Name, archive_ext)
		}
	}

	ds := Dataset{
		Name:                   conf.Name,
		archive_ext:            archive_ext,
		download_dir:           conf.DownloadDir,
		download_dir_exists:    false,
		download_headers:       conf.DownloadHeaders,
		get_download_file_time: conf.GetDownloadFileTime,
		is_stale:               conf.IsStale,
		log:                    conf.Log,
		prefix_time_format:     "2006-01-02-15",
		url:                    conf.URL,
	}
	return &ds
}

func (ds *Dataset) ensureDownloadDir() error {
	if !ds.download_dir_exists {
		err := EnsureDir(ds.download_dir)
		if err != nil {
			return err
		}
		ds.download_dir_exists = true
	}
	return nil
}

func (ds Dataset) list(includeArchives bool) ([]string, error) {
	filenames, err := fs.Glob(os.DirFS(ds.download_dir), "*-"+ds.Name)
	if err != nil {
		return nil, err
	}
	if includeArchives && ds.archive_ext != "" {
		archiveFilenames, err := fs.Glob(os.DirFS(ds.download_dir), "*-"+ds.Name+ds.archive_ext)
		if err != nil {
			return nil, err
		}
		filenames = append(filenames, archiveFilenames...)
	}
	return filenames, nil
}

func (ds Dataset) parseTime(filename string) (time.Time, error) {
	t, err := time.Parse(ds.prefix_time_format, filename[0:len(ds.prefix_time_format)])
	if err != nil {
		ds.removeFile(filename)
		if ds.archive_ext != "" {
			ds.removeFile(filename + ds.archive_ext)
		}
	}
	return t, err
}

func (ds Dataset) isStale(filename string) bool {
	if filename == "" {
		return true
	}
	fileDate, err := ds.parseTime(filename)
	if err != nil {
		ds.log.Error("failed to parse date", "filename", filename, "error", err)
		return true
	}
	return ds.is_stale(fileDate)

}

func (ds *Dataset) prepareFilenames() error {
	if ds.prev_filename != "" {
		return nil
	}
	filenames, err := ds.list(false)
	if err != nil {
		return aError{"failed to list dataset files", err}
	}
	slices.SortFunc(filenames, func(a, b string) int {
		// asc
		aTime, aErr := ds.parseTime(a)
		if aErr != nil {
			ds.log.Error("failed to parse date", "filename", a, "error", aErr)
			return 0
		}
		bTime, bErr := ds.parseTime(b)
		if bErr != nil {
			ds.log.Error("failed to parse date", "filename", b, "error", bErr)
			return 0
		}
		return aTime.Compare(bTime)
	})

	lastFilename := ""
	if len(filenames) > 0 {
		lastFilename = filenames[len(filenames)-1]
	}
	if ds.isStale(lastFilename) {
		ds.curr_filename = ds.get_download_file_time().Format(ds.prefix_time_format) + "-" + ds.Name
		ds.prev_filename = lastFilename
	} else {
		ds.curr_filename = lastFilename
		if len(filenames) > 1 {
			ds.prev_filename = filenames[len(filenames)-2]
		} else {
			ds.prev_filename = ""
		}
	}
	if exists, err := FileExists(ds.filePath(ds.curr_filename)); err != nil || !exists {
		ds.curr_filename = ds.get_download_file_time().Format(ds.prefix_time_format) + "-" + ds.Name
	}
	if ds.prev_filename != "" {
		if exists, err := FileExists(ds.filePath(ds.prev_filename)); err != nil || !exists {
			ds.prev_filename = ""
		}
	}
	return nil
}

func (ds Dataset) removeFile(filename string) {
	if err := os.Remove(path.Join(ds.download_dir, filename)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		ds.log.Warn("failed to remove file", "filename", filename, "error", err)
	}
}

func (ds Dataset) cleanup() error {
	filenames, err := ds.list(true)
	if err != nil {
		return aError{"failed to list dataset files", err}
	}
	for _, filename := range filenames {
		if strings.HasPrefix(filename, ds.curr_filename) || strings.HasPrefix(filename, ds.prev_filename) {
			continue
		}
		ds.removeFile(filename)
	}
	return nil
}

func (ds Dataset) filePath(fileName string) string {
	return path.Join(ds.download_dir, fileName)
}

func (ds *Dataset) download() error {
	filename := ds.curr_filename
	filePath := ds.filePath(filename)
	if exists, err := FileExists(filePath); err != nil {
		return aError{"failed to check existing file", err}
	} else if exists {
		ds.log.Info("found already downloaded", "filename", filepath.Base(filePath))
		return nil
	}

	dlFilePath := filePath + ds.archive_ext
	dlFilename := filepath.Base(dlFilePath)
	if exists, err := FileExists(dlFilePath); err != nil {
		return aError{"failed to check existing file", err}
	} else if !exists {
		ds.log.Info("downloading...", "filename", dlFilename)
		req, err := http.NewRequest("GET", ds.url, nil)
		if err != nil {
			return err
		}
		for k, v := range ds.download_headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		out, err := os.Create(dlFilePath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
		ds.log.Info("downloaded", "filename", dlFilename)
	} else {
		ds.log.Info("found already downloaded", "filename", dlFilename)
	}

	if ds.archive_ext == "" {
		return nil
	}

	if ds.archive_ext == ".gz" {
		ds.log.Info("extracting...", "filename", filename)
		f, err := os.Open(dlFilePath)
		if err != nil {
			return err
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gzr.Close()

		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, gzr)
		if err != nil {
			return nil
		}
		ds.log.Info("extracted", "filename", filename)

		ds.removeFile(dlFilename)
	}

	return nil
}

func (ds *Dataset) Init() error {
	if err := ds.ensureDownloadDir(); err != nil {
		return err
	}

	if err := ds.prepareFilenames(); err != nil {
		return err
	}

	if ds.prev_filename != "" {
		ds.log.Info("found existing", "filename", ds.prev_filename)
		if err := ds.cleanup(); err != nil {
			return err
		}
	}

	if err := ds.download(); err != nil {
		return err
	}

	return nil
}
