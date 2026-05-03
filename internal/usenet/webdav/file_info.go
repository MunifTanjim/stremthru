package usenet_webdav

import (
	"os"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
)

type rootDirInfo struct{}

func (d *rootDirInfo) Name() string       { return "/" }
func (d *rootDirInfo) Size() int64        { return 0 }
func (d *rootDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (d *rootDirInfo) ModTime() time.Time { return time.Now() }
func (d *rootDirInfo) IsDir() bool        { return true }
func (d *rootDirInfo) Sys() any           { return nil }

type nzbDirInfo struct {
	info *nzb_info.NZBInfo
}

func (d *nzbDirInfo) Name() string {
	return stripNZBExtension(d.info.Name)
}
func (d *nzbDirInfo) Size() int64        { return 0 }
func (d *nzbDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (d *nzbDirInfo) ModTime() time.Time { return d.info.UAt.Time }
func (d *nzbDirInfo) IsDir() bool        { return true }
func (d *nzbDirInfo) Sys() any           { return nil }

type contentEntryInfo struct {
	entry *ContentEntry
}

func (e *contentEntryInfo) Name() string       { return e.entry.Name }
func (e *contentEntryInfo) Size() int64        { return e.entry.Size }
func (e *contentEntryInfo) Mode() os.FileMode  { return 0644 }
func (e *contentEntryInfo) ModTime() time.Time { return e.entry.ModTime }
func (e *contentEntryInfo) IsDir() bool        { return false }
func (e *contentEntryInfo) Sys() any           { return nil }
