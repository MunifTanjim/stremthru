package usenet_pool

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/spf13/afero"
)

var (
	_ fs.FS       = (*UsenetFS)(nil)
	_ fs.File     = (*UsenetFile)(nil)
	_ fs.FileInfo = (*UsenetFileInfo)(nil)
	_ afero.Fs    = (*UsenetFSAfero)(nil)
	_ afero.File  = (*UsenetFileAfero)(nil)
)

type UsenetFileInfo struct {
	f    *nzb.File
	size int64
}

func (ufi *UsenetFileInfo) Name() string       { return ufi.f.GetName() }
func (ufi *UsenetFileInfo) Size() int64        { return ufi.size }
func (ufi *UsenetFileInfo) Mode() fs.FileMode  { return 0644 }
func (ufi *UsenetFileInfo) ModTime() time.Time { return time.Unix(ufi.f.Date, 0) }
func (ufi *UsenetFileInfo) IsDir() bool        { return false }
func (ufi *UsenetFileInfo) Sys() any           { return nil }

type UsenetFS struct {
	ctx               context.Context
	pool              *Pool
	nzb               *nzb.NZB
	files             map[string]UsenetFileInfo
	segmentBufferSize int64
}

type UsenetFSConfig struct {
	NZB               *nzb.NZB
	Pool              *Pool
	SegmentBufferSize int64
}

func NewUsenetFS(ctx context.Context, conf *UsenetFSConfig) *UsenetFS {
	usenetFs := &UsenetFS{
		ctx:               ctx,
		pool:              conf.Pool,
		nzb:               conf.NZB,
		files:             make(map[string]UsenetFileInfo, conf.NZB.FileCount()),
		segmentBufferSize: conf.SegmentBufferSize,
	}
	for i := range conf.NZB.Files {
		f := &conf.NZB.Files[i]
		usenetFs.files[f.GetName()] = UsenetFileInfo{
			f: f,
		}
	}
	return usenetFs
}

func (ufs *UsenetFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)

	fi, ok := ufs.files[name]
	if !ok {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	stream, err := NewFileStream(ufs.pool, fi.f, ufs.segmentBufferSize)
	if err != nil {
		return nil, err
	}

	fi.size = stream.Size()

	return &UsenetFile{
		FileStream: stream,
		fi:         &fi,
	}, nil
}

func (ufs *UsenetFS) Stat(name string) (os.FileInfo, error) {
	name = path.Clean(name)

	fi, ok := ufs.files[name]
	if !ok {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	firstSegment, err := ufs.pool.fetchFirstSegment(ufs.ctx, fi.f)
	if err != nil {
		return nil, err
	}
	fi.size = firstSegment.FileSize

	return &fi, nil
}

func (ufs *UsenetFS) toAfero() *UsenetFSAfero {
	return &UsenetFSAfero{ufs}
}

type UsenetFile struct {
	*FileStream
	fi *UsenetFileInfo
}

func (uf *UsenetFile) Stat() (fs.FileInfo, error) {
	return uf.fi, nil
}

type UsenetFSAfero struct {
	*UsenetFS
}

func (u *UsenetFSAfero) Chmod(name string, mode os.FileMode) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) Chown(name string, uid int, gid int) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) Create(name string) (afero.File, error) {
	return nil, syscall.EPERM
}

func (u *UsenetFSAfero) Mkdir(name string, perm os.FileMode) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) MkdirAll(path string, perm os.FileMode) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) Name() string {
	return "UsenetFsAfero"
}

func (u *UsenetFSAfero) Open(name string) (afero.File, error) {
	f, err := u.UsenetFS.Open(name)
	if err != nil {
		return nil, err
	}
	return &UsenetFileAfero{f.(*UsenetFile)}, nil
}

func (u *UsenetFSAfero) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	return u.Open(name)
}

func (u *UsenetFSAfero) Remove(name string) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) RemoveAll(path string) error {
	return syscall.EPERM
}

func (u *UsenetFSAfero) Rename(oldname string, newname string) error {
	return syscall.EPERM
}

type UsenetFileAfero struct {
	*UsenetFile
}

func (u *UsenetFileAfero) Name() string {
	return u.fi.Name()
}

func (u *UsenetFileAfero) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not supported")
}

func (u *UsenetFileAfero) Readdirnames(n int) ([]string, error) {
	return nil, errors.New("not supported")
}

func (u *UsenetFileAfero) Sync() error {
	return nil
}

func (u *UsenetFileAfero) Truncate(size int64) error {
	return syscall.EPERM
}

func (u *UsenetFileAfero) Write(p []byte) (n int, err error) {
	return 0, syscall.EPERM
}

func (u *UsenetFileAfero) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EPERM
}

func (u *UsenetFileAfero) WriteString(s string) (ret int, err error) {
	return 0, syscall.EPERM
}
