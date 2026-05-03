package usenet_webdav

import (
	"context"
	"os"
	"path"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	"golang.org/x/net/webdav"
)

var _ webdav.FileSystem = (*FileSystem)(nil)

type FileSystem struct{}

func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return os.ErrPermission
}

func (fs *FileSystem) RemoveAll(ctx context.Context, name string) error {
	return os.ErrPermission
}

func (fs *FileSystem) Rename(ctx context.Context, oldName, newName string) error {
	return os.ErrPermission
}

func (fs *FileSystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, os.ErrPermission
	}
	return fs.open(ctx, name)
}

func (fs *FileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean("/" + name)

	if name == "/" {
		return &rootDirInfo{}, nil
	}

	parts := splitPath(name)
	if len(parts) == 0 {
		return nil, pathError("stat", name, os.ErrNotExist)
	}

	nzbName := parts[0]
	info, err := fs.findNZBInfoByName(nzbName)
	if err != nil {
		return nil, pathError("stat", name, err)
	}
	if info == nil {
		return nil, pathError("stat", name, os.ErrNotExist)
	}

	if len(parts) == 1 {
		return &nzbDirInfo{info: info}, nil
	}

	if len(parts) > 2 {
		return nil, pathError("stat", name, os.ErrNotExist)
	}

	// Find entry by name (flat structure)
	entries := TransformContentFiles(info.ContentFiles.Data, info.UAt.Time)
	entry := findContentEntry(entries, parts[1])
	if entry == nil {
		return nil, pathError("stat", name, os.ErrNotExist)
	}

	return &contentEntryInfo{entry: entry}, nil
}

func (fs *FileSystem) open(ctx context.Context, name string) (webdav.File, error) {
	name = path.Clean("/" + name)

	if name == "/" {
		return fs.openRootDir(ctx)
	}

	parts := splitPath(name)
	if len(parts) == 0 {
		return nil, pathError("open", name, os.ErrNotExist)
	}

	nzbName := parts[0]
	info, err := fs.findNZBInfoByName(nzbName)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	if info == nil {
		return nil, pathError("open", name, os.ErrNotExist)
	}

	if len(parts) == 1 {
		return fs.openNZBDir(ctx, info)
	}

	if len(parts) > 2 {
		return nil, pathError("open", name, os.ErrNotExist)
	}

	// Find entry by name (flat structure)
	entries := TransformContentFiles(info.ContentFiles.Data, info.UAt.Time)
	entry := findContentEntry(entries, parts[1])
	if entry == nil {
		return nil, pathError("open", name, os.ErrNotExist)
	}

	return fs.openContentFile(ctx, info, entry)
}

func (fs *FileSystem) findNZBInfoByName(name string) (*nzb_info.NZBInfo, error) {
	infos, err := nzb_info.GetAll()
	if err != nil {
		return nil, err
	}
	for i := range infos {
		info := &infos[i]
		if stripNZBExtension(info.Name) == name && info.Status == statusDownloaded && nzb_info.IsNZBFileCached(info.Hash) {
			return info, nil
		}
	}
	return nil, nil
}

func (fs *FileSystem) openRootDir(ctx context.Context) (webdav.File, error) {
	infos, err := nzb_info.GetAll()
	if err != nil {
		return nil, err
	}

	entries := make([]os.FileInfo, 0, len(infos))
	for i := range infos {
		info := &infos[i]
		if info.Status == statusDownloaded && nzb_info.IsNZBFileCached(info.Hash) {
			entries = append(entries, &nzbDirInfo{info: info})
		}
	}

	return &webdavDir{
		info:    &rootDirInfo{},
		entries: entries,
	}, nil
}

func (fs *FileSystem) openNZBDir(ctx context.Context, info *nzb_info.NZBInfo) (webdav.File, error) {
	entries := TransformContentFiles(info.ContentFiles.Data, info.UAt.Time)

	fileInfos := make([]os.FileInfo, 0, len(entries))
	for i := range entries {
		fileInfos = append(fileInfos, &contentEntryInfo{entry: &entries[i]})
	}

	return &webdavDir{
		info:    &nzbDirInfo{info: info},
		entries: fileInfos,
	}, nil
}

func (fs *FileSystem) openContentFile(ctx context.Context, info *nzb_info.NZBInfo, entry *ContentEntry) (webdav.File, error) {
	return &webdavFile{
		info:        &contentEntryInfo{entry: entry},
		nzbInfo:     info,
		contentPath: entry.ContentPath,
		ctx:         ctx,
	}, nil
}
