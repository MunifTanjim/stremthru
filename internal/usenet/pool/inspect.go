package usenet_pool

import (
	"context"
	"path/filepath"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var inspectLog = logger.Scoped("usenet/pool/inspect")

type NZBContentFileType string

const (
	NZBContentFileTypeVideo   NZBContentFileType = "video"
	NZBContentFileTypeArchive NZBContentFileType = "archive"
	NZBContentFileTypeOther   NZBContentFileType = "other"
)

type NZBContentFile struct {
	Type       NZBContentFileType `json:"t"`
	Name       string             `json:"n"`
	Size       int64              `json:"s"`
	Streamable bool               `json:"strm"`
	Files      []NZBContentFile   `json:"files,omitempty"`
	Parts      []NZBContentFile   `json:"parts,omitempty"`
}

type NZBContent struct {
	Files      []NZBContentFile
	Streamable bool
}

func classifyNZBContentFileType(filename string) NZBContentFileType {
	if isVideoFile(filename) {
		return NZBContentFileTypeVideo
	}
	if IsArchiveFile(filename) {
		return NZBContentFileTypeArchive
	}
	return NZBContentFileTypeOther
}

func areNZBContentFilesStreamable(files []NZBContentFile) bool {
	for i := range files {
		f := &files[i]
		if !f.Streamable {
			return false
		}
		if len(f.Files) > 0 && !areNZBContentFilesStreamable(f.Files) {
			return false
		}
	}
	return true
}

func isNZBStremable(c *NZBContent) bool {
	return areNZBContentFilesStreamable(c.Files)
}

func (p *Pool) InspectNZBContent(ctx context.Context, nzbDoc *nzb.NZB, password string) (*NZBContent, error) {
	content := &NZBContent{
		Files:      []NZBContentFile{},
		Streamable: true,
	}

	if len(nzbDoc.Files) == 0 {
		return content, nil
	}

	var nzbArchiveFiles []*nzb.File

	for i := range nzbDoc.Files {
		f := &nzbDoc.Files[i]
		filename := f.Name()

		if isVideoFile(filename) {
			content.Files = append(content.Files, NZBContentFile{
				Type:       NZBContentFileTypeVideo,
				Name:       filename,
				Size:       f.Size(),
				Streamable: true,
			})
			continue
		}

		if IsArchiveFile(filename) {
			nzbArchiveFiles = append(nzbArchiveFiles, f)
			continue
		}

		content.Files = append(content.Files, NZBContentFile{
			Type:       NZBContentFileTypeOther,
			Name:       filename,
			Size:       f.Size(),
			Streamable: true,
		})
	}

	archiveGroups := groupArchiveVolumes(nzbArchiveFiles)

	for i := range archiveGroups {
		group := &archiveGroups[i]
		name := group.Files[0].Name()

		entry := NZBContentFile{
			Type: NZBContentFileTypeArchive,
			Name: name,
			Size: group.TotalSize,
		}
		for _, f := range group.Files {
			entry.Parts = append(entry.Parts, NZBContentFile{
				Type:       classifyNZBContentFileType(f.Name()),
				Name:       f.Name(),
				Size:       f.Size(),
				Streamable: true,
			})
		}

		ufs := NewUsenetFS(ctx, &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: p,
		})

		var archive Archive
		switch group.FileType {
		case FileTypeRAR:
			archive = NewUsenetRARArchive(ufs)
		case FileType7z:
			archive = NewUsenetSevenZipArchive(ufs)
		}

		if err := archive.Open(password); err != nil {
			inspectLog.Warn("failed to open archive", "error", err, "name", name)
			content.Files = append(content.Files, entry)
			continue
		}

		entry.Streamable = archive.IsStreamable()
		if entry.Streamable {
			files, err := archive.GetFiles()
			if err != nil {
				inspectLog.Warn("failed to get archive files", "name", name, "error", err)
			} else {
				entry.Files = p.inspectArchiveFiles(files, password)
			}
		}

		archive.Close()
		content.Files = append(content.Files, entry)
	}

	content.Streamable = isNZBStremable(content)

	return content, nil
}

func (p *Pool) inspectArchiveFiles(files []ArchiveFile, password string) []NZBContentFile {
	archiveGroups := groupArchiveVolumes(files)

	if len(archiveGroups) == 0 {
		result := make([]NZBContentFile, len(files))
		for i, f := range files {
			result[i] = NZBContentFile{
				Type:       classifyNZBContentFileType(f.Name()),
				Name:       f.Name(),
				Size:       f.Size(),
				Streamable: f.IsStreamable(),
			}
		}
		return result
	}

	archiveFileNames := make(map[string]struct{})
	for i := range archiveGroups {
		for _, f := range archiveGroups[i].Files {
			archiveFileNames[f.Name()] = struct{}{}
		}
	}

	var result []NZBContentFile

	for _, f := range files {
		if _, isArchivePart := archiveFileNames[f.Name()]; !isArchivePart {
			result = append(result, NZBContentFile{
				Type:       classifyNZBContentFileType(f.Name()),
				Name:       f.Name(),
				Size:       f.Size(),
				Streamable: f.IsStreamable(),
			})
		}
	}

	for i := range archiveGroups {
		group := &archiveGroups[i]
		name := group.Files[0].Name()

		entry := NZBContentFile{
			Type: NZBContentFileTypeArchive,
			Name: name,
			Size: group.TotalSize,
		}
		for _, f := range group.Files {
			entry.Parts = append(entry.Parts, NZBContentFile{
				Type:       classifyNZBContentFileType(f.Name()),
				Name:       f.Name(),
				Size:       f.Size(),
				Streamable: true,
			})
		}

		allStreamable := true
		for _, f := range group.Files {
			if !f.IsStreamable() {
				allStreamable = false
				break
			}
		}

		if !allStreamable {
			result = append(result, entry)
			continue
		}

		afs := NewArchiveFS(group.Files)

		var innerArchive Archive
		switch group.FileType {
		case FileTypeRAR:
			innerArchive = NewRARArchive(afs, filepath.Base(name))
		case FileType7z:
			innerArchive = NewSevenZipArchive(afs.toAfero(), filepath.Base(name))
		default:
			afs.Close()
			result = append(result, entry)
			continue
		}

		if err := innerArchive.Open(""); err != nil {
			inspectLog.Warn("failed to open nested archive", "error", err, "name", name)
			afs.Close()
			result = append(result, entry)
			continue
		}

		entry.Streamable = innerArchive.IsStreamable()
		if entry.Streamable {
			if innerFiles, err := innerArchive.GetFiles(); err != nil {
				inspectLog.Warn("failed to get nested archive files", "error", err, "name", name)
			} else {
				innerContentFiles := make([]NZBContentFile, len(innerFiles))
				for j, f := range innerFiles {
					innerContentFiles[j] = NZBContentFile{
						Type:       classifyNZBContentFileType(f.Name()),
						Name:       f.Name(),
						Size:       f.Size(),
						Streamable: f.IsStreamable(),
					}
				}
				entry.Files = innerContentFiles
			}
		}

		innerArchive.Close()
		result = append(result, entry)
	}

	return result
}
