package usenet_pool

import (
	"context"
	"io/fs"
	"path"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var (
	_ fs.FS       = (*UsenetFS)(nil)
	_ fs.File     = (*UsenetFile)(nil)
	_ fs.FileInfo = (*UsenetFileInfo)(nil)
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
	ctx           context.Context
	pool          *Pool
	cache         *SegmentCache
	nzb           *nzb.NZB
	files         map[string]UsenetFileInfo
	segmentBuffer int
}

type UsenetFSConfig struct {
	NZB           *nzb.NZB
	Pool          *Pool
	Cache         *SegmentCache
	SegmentBuffer int
}

func (conf *UsenetFSConfig) setDefaults() {
	if conf.SegmentBuffer <= 0 {
		conf.SegmentBuffer = 5
	}
}

func NewUsenetFS(ctx context.Context, conf *UsenetFSConfig) *UsenetFS {
	conf.setDefaults()

	usenetFs := &UsenetFS{
		ctx:           ctx,
		pool:          conf.Pool,
		cache:         conf.Cache,
		nzb:           conf.NZB,
		files:         make(map[string]UsenetFileInfo, conf.NZB.FileCount()),
		segmentBuffer: conf.SegmentBuffer,
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

	firstSegment, err := ufs.pool.fetchFirstSegment(ufs.ctx, fi.f, ufs.cache)
	if err != nil {
		return nil, err
	}
	fi.size = firstSegment.Header.FileSize
	println("file size", fi.size)

	return &UsenetFile{
		FileStream: NewFileStream(ufs.ctx, ufs.pool, fi.f, fi.size, ufs.segmentBuffer, ufs.cache),
		fi:         &fi,
	}, nil
}

type UsenetFile struct {
	*FileStream
	fi  *UsenetFileInfo
	ufs *UsenetFS
}

func (uf *UsenetFile) Stat() (fs.FileInfo, error) {
	return uf.fi, nil
}
