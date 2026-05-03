package usenet_webdav

import (
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	usenet_manager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"golang.org/x/net/webdav"
)

func getPool() (*usenet_pool.Pool, error) {
	return usenet_manager.GetPool()
}

var (
	_ webdav.File = (*webdavDir)(nil)
	_ webdav.File = (*webdavFile)(nil)
)

type webdavDir struct {
	info    os.FileInfo
	entries []os.FileInfo
	offset  int
}

func (d *webdavDir) Close() error {
	return nil
}

func (d *webdavDir) Read(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}

func (d *webdavDir) Seek(offset int64, whence int) (int64, error) {
	return 0, os.ErrInvalid
}

func (d *webdavDir) Readdir(count int) ([]os.FileInfo, error) {
	if count <= 0 {
		entries := d.entries[d.offset:]
		d.offset = len(d.entries)
		return entries, nil
	}

	if d.offset >= len(d.entries) {
		return nil, io.EOF
	}

	end := d.offset + count
	if end > len(d.entries) {
		end = len(d.entries)
	}

	entries := d.entries[d.offset:end]
	d.offset = end
	return entries, nil
}

func (d *webdavDir) Stat() (os.FileInfo, error) {
	return d.info, nil
}

func (d *webdavDir) Write(p []byte) (n int, err error) {
	return 0, os.ErrPermission
}

type webdavFile struct {
	info        os.FileInfo
	nzbInfo     *nzb_info.NZBInfo
	contentPath string
	ctx         context.Context

	initOnce sync.Once
	initErr  error
	stream   *usenet_pool.Stream
}

func (f *webdavFile) init() error {
	f.initOnce.Do(func() {
		nzbFile := nzb_info.GetCachedNZBFile(f.nzbInfo.Hash)
		if nzbFile == nil {
			nzbFile, f.initErr = nzb_info.FetchNZBFile(f.nzbInfo.URL, f.nzbInfo.Name, nil)
			if f.initErr != nil {
				return
			}
		}

		nzbDoc, err := nzb.ParseBytes(nzbFile.Blob)
		if err != nil {
			f.initErr = err
			return
		}

		pool, err := getPool()
		if err != nil {
			f.initErr = err
			return
		}

		webdavContentPath := strings.ReplaceAll(f.contentPath, "/", "::")

		f.stream, f.initErr = pool.StreamByContentPath(f.ctx, nzbDoc, webdavContentPath, &usenet_pool.StreamConfig{
			Password:     f.nzbInfo.Password,
			ContentFiles: f.nzbInfo.ContentFiles.Data,
		})
	})
	return f.initErr
}

func (f *webdavFile) Close() error {
	if f.stream != nil {
		return f.stream.Close()
	}
	return nil
}

func (f *webdavFile) Read(p []byte) (n int, err error) {
	if err := f.ctx.Err(); err != nil {
		return 0, err
	}
	if err := f.init(); err != nil {
		return 0, err
	}
	return f.stream.Read(p)
}

func (f *webdavFile) Seek(offset int64, whence int) (int64, error) {
	if err := f.ctx.Err(); err != nil {
		return 0, err
	}
	if err := f.init(); err != nil {
		return 0, err
	}
	return f.stream.Seek(offset, whence)
}

func (f *webdavFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (f *webdavFile) Stat() (os.FileInfo, error) {
	return f.info, nil
}

func (f *webdavFile) Write(p []byte) (n int, err error) {
	return 0, os.ErrPermission
}
