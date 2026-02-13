package usenet_pool

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

type Archive interface {
	Open(password string) error
	Close() error
	GetFiles() ([]ArchiveFile, error)
	IsStreamable() bool
}

type ArchiveFile interface {
	Name() string
	Size() int64
	PackedSize() int64
	IsStreamable() bool
	Open() (io.ReadSeekCloser, error)
}

var (
	_ fs.FS       = (*ArchiveFS)(nil)
	_ fs.File     = (*ArchiveVirtualFile)(nil)
	_ fs.FileInfo = (*ArchiveVirtualFileInfo)(nil)
	_ afero.Fs    = (*ArchiveFSAfero)(nil)
	_ afero.File  = (*ArchiveVirtualFileAfero)(nil)
)

type ArchiveFS struct {
	files         map[string]ArchiveFile // filename -> ArchiveFile
	mu            sync.Mutex
	openedReaders []io.Closer
}

func NewArchiveFS(files []ArchiveFile) *ArchiveFS {
	fileMap := make(map[string]ArchiveFile, len(files))
	for _, f := range files {
		name := path.Base(f.Name())
		fileMap[name] = f
	}
	return &ArchiveFS{
		files:         fileMap,
		openedReaders: make([]io.Closer, 0),
	}
}

func (afs *ArchiveFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	if name == "." || name == "/" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	af, ok := afs.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	r, err := af.Open()
	if err != nil {
		return nil, err
	}

	afs.mu.Lock()
	afs.openedReaders = append(afs.openedReaders, r)
	afs.mu.Unlock()

	return &ArchiveVirtualFile{
		ReadSeekCloser: r,
		name:           name,
		size:           af.Size(),
	}, nil
}

func (afs *ArchiveFS) Stat(name string) (os.FileInfo, error) {
	name = path.Clean(name)

	af, ok := afs.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	return &ArchiveVirtualFileInfo{
		name: name,
		size: af.Size(),
	}, nil
}

func (afs *ArchiveFS) Close() error {
	afs.mu.Lock()
	defer afs.mu.Unlock()

	var errs []error
	for _, r := range afs.openedReaders {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	afs.openedReaders = nil

	return errors.Join(errs...)
}

func (afs *ArchiveFS) toAfero() *ArchiveFSAfero {
	return &ArchiveFSAfero{afs}
}

type ArchiveVirtualFile struct {
	io.ReadSeekCloser
	name string
	size int64
}

func (avf *ArchiveVirtualFile) Stat() (fs.FileInfo, error) {
	return &ArchiveVirtualFileInfo{
		name: avf.name,
		size: avf.size,
	}, nil
}

type ArchiveVirtualFileInfo struct {
	name string
	size int64
}

func (fi *ArchiveVirtualFileInfo) Name() string       { return fi.name }
func (fi *ArchiveVirtualFileInfo) Size() int64        { return fi.size }
func (fi *ArchiveVirtualFileInfo) Mode() fs.FileMode  { return 0644 }
func (fi *ArchiveVirtualFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *ArchiveVirtualFileInfo) IsDir() bool        { return false }
func (fi *ArchiveVirtualFileInfo) Sys() any           { return nil }

type ArchiveFSAfero struct {
	*ArchiveFS
}

func (a *ArchiveFSAfero) Name() string { return "ArchiveFsAfero" }

func (a *ArchiveFSAfero) Open(name string) (afero.File, error) {
	f, err := a.ArchiveFS.Open(name)
	if err != nil {
		return nil, err
	}
	return &ArchiveVirtualFileAfero{f.(*ArchiveVirtualFile)}, nil
}

func (a *ArchiveFSAfero) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|syscall.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, syscall.EPERM
	}
	return a.Open(name)
}

func (a *ArchiveFSAfero) Chmod(name string, mode os.FileMode) error         { return syscall.EPERM }
func (a *ArchiveFSAfero) Chown(name string, uid int, gid int) error         { return syscall.EPERM }
func (a *ArchiveFSAfero) Chtimes(name string, atime, mtime time.Time) error { return syscall.EPERM }
func (a *ArchiveFSAfero) Create(name string) (afero.File, error)            { return nil, syscall.EPERM }
func (a *ArchiveFSAfero) Mkdir(name string, perm os.FileMode) error         { return syscall.EPERM }
func (a *ArchiveFSAfero) MkdirAll(path string, perm os.FileMode) error      { return syscall.EPERM }
func (a *ArchiveFSAfero) Remove(name string) error                          { return syscall.EPERM }
func (a *ArchiveFSAfero) RemoveAll(path string) error                       { return syscall.EPERM }
func (a *ArchiveFSAfero) Rename(oldname, newname string) error              { return syscall.EPERM }

type ArchiveVirtualFileAfero struct {
	*ArchiveVirtualFile
}

func (a *ArchiveVirtualFileAfero) Name() string { return a.name }
func (a *ArchiveVirtualFileAfero) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not supported")
}
func (a *ArchiveVirtualFileAfero) Readdirnames(n int) ([]string, error) {
	return nil, errors.New("not supported")
}
func (a *ArchiveVirtualFileAfero) Sync() error                       { return nil }
func (a *ArchiveVirtualFileAfero) Truncate(size int64) error         { return syscall.EPERM }
func (a *ArchiveVirtualFileAfero) Write(p []byte) (n int, err error) { return 0, syscall.EPERM }
func (a *ArchiveVirtualFileAfero) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EPERM
}
func (a *ArchiveVirtualFileAfero) WriteString(s string) (ret int, err error) { return 0, syscall.EPERM }

func (a *ArchiveVirtualFileAfero) ReadAt(p []byte, off int64) (n int, err error) {
	if ra, ok := a.ReadSeekCloser.(io.ReaderAt); ok {
		return ra.ReadAt(p, off)
	}

	currentPos, err := a.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	_, err = a.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}

	n, readErr := io.ReadFull(a.ReadSeekCloser, p)

	_, seekErr := a.Seek(currentPos, io.SeekStart)
	if seekErr != nil && readErr == nil {
		return n, seekErr
	}

	if readErr == io.ErrUnexpectedEOF {
		readErr = io.EOF
	}

	return n, readErr
}
