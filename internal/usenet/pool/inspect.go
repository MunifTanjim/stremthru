package usenet_pool

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
	"github.com/nwaples/rardecode/v2"
)

var inspectLog = logger.Scoped("usenet/pool/inspect")

type NZBContentFileType string

const (
	NZBContentFileTypeVideo   NZBContentFileType = "video"
	NZBContentFileTypeArchive NZBContentFileType = "archive"
	NZBContentFileTypeOther   NZBContentFileType = "other"
	NZBContentFileTypeUnknown NZBContentFileType = ""
)

const (
	NZBContentFileErrorArticleNotFound = "article_not_found"
	NZBContentFileErrorMissingPassword = "missing_password"
	NZBContentFileErrorOpenFailed      = "open_failed"
)

type NZBContentFile struct {
	Type       NZBContentFileType `json:"t"`
	Name       string             `json:"n"`
	Alias      string             `json:"alias,omitempty"`
	Size       int64              `json:"s"`
	Volume     int                `json:"vol,omitempty"`
	Streamable bool               `json:"strm"`
	Errors     []string           `json:"errs,omitempty"`
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

func hasStreamableVideoInNZBContentFiles(files []NZBContentFile) bool {
	for i := range files {
		f := &files[i]
		name := f.Name
		if f.Alias != "" {
			name = f.Alias
		}
		isVideo := isVideoFile(name)
		if f.Streamable && isVideo {
			return true
		}
		if len(f.Files) > 0 && hasStreamableVideoInNZBContentFiles(f.Files) {
			return true
		}
	}
	return false
}

func isNZBStremable(c *NZBContent) bool {
	return hasStreamableVideoInNZBContentFiles(c.Files)
}

type nzbArchiveFile struct {
	filetype FileType
	name     string
	size     int64
	volume   int
}

func (f *nzbArchiveFile) Name() string {
	return f.name
}

func (f *nzbArchiveFile) Size() int64 {
	return f.size
}

func (f *nzbArchiveFile) FileType() FileType {
	return f.filetype
}

func (f *nzbArchiveFile) Volume() int {
	if f.volume >= 0 {
		return f.volume
	}
	switch f.filetype {
	case FileTypeRAR:
		return GetRARVolumeNumber(f.Name())
	case FileType7z:
		return Get7zVolumeNumber(f.Name())
	default:
		return -1
	}
}

func (p *Pool) InspectNZBContent(ctx context.Context, nzbDoc *nzb.NZB, password string) (*NZBContent, error) {
	content := &NZBContent{
		Files:      []NZBContentFile{},
		Streamable: true,
	}

	if len(nzbDoc.Files) == 0 {
		return content, nil
	}

	var nzbArchiveFiles []*nzbArchiveFile

	type segmentFetchResult struct {
		nzbFile      *nzb.File
		startSegment *SegmentData
		startErr     error
		endSegment   *SegmentData
		endErr       error
	}

	var needsFetch []*nzb.File

	for i := range nzbDoc.Files {
		f := &nzbDoc.Files[i]

		if f.SegmentCount() == 0 {
			content.Files = append(content.Files, NZBContentFile{
				Type:       NZBContentFileTypeOther,
				Name:       f.Name(),
				Size:       f.Size(),
				Streamable: false,
			})
			continue
		}

		needsFetch = append(needsFetch, f)
	}

	fetchResults := make([]segmentFetchResult, len(needsFetch))

	type segmentFetchTask struct {
		fileIdx int
		segment *nzb.Segment
		groups  []string
		isEnd   bool
	}

	var tasks []segmentFetchTask
	for i, f := range needsFetch {
		tasks = append(tasks, segmentFetchTask{fileIdx: i, segment: &f.Segments[0], groups: f.Groups, isEnd: false})
		if f.SegmentCount() > 1 {
			tasks = append(tasks, segmentFetchTask{fileIdx: i, segment: &f.Segments[len(f.Segments)-1], groups: f.Groups, isEnd: true})
		}
	}

	taskResults := make([]struct {
		data *SegmentData
		err  error
	}, len(tasks))
	fetchPool := pond.NewPool(config.Newz.MaxConnectionPerStream)
	for i, t := range tasks {
		fetchPool.Submit(func() {
			taskResults[i].data, taskResults[i].err = p.fetchSegment(ctx, t.segment, t.groups)
		})
	}
	fetchPool.StopAndWait()

	for i, t := range tasks {
		if t.isEnd {
			fetchResults[t.fileIdx].endSegment = taskResults[i].data
			fetchResults[t.fileIdx].endErr = taskResults[i].err
		} else {
			fetchResults[t.fileIdx].startSegment = taskResults[i].data
			fetchResults[t.fileIdx].startErr = taskResults[i].err
			fetchResults[t.fileIdx].nzbFile = needsFetch[t.fileIdx]
		}
	}

	for _, fr := range fetchResults {
		filename := fr.nzbFile.Name()

		articleNotFound := errors.Is(fr.startErr, ErrArticleNotFound) || errors.Is(fr.endErr, ErrArticleNotFound)

		var firstBytes []byte
		if fr.startSegment != nil {
			firstBytes = fr.startSegment.Body
		}
		detectedType := DetectFileType(firstBytes, filename)

		isArchive := detectedType == FileTypeRAR || detectedType == FileType7z
		isVideoByExt := isVideoFile(filename)

		entry := NZBContentFile{
			Name:       filename,
			Size:       fr.nzbFile.Size(),
			Streamable: true,
		}
		if isArchive {
			entry.Type = NZBContentFileTypeArchive
		} else if isVideoByExt {
			entry.Type = NZBContentFileTypeVideo
		} else {
			entry.Type = NZBContentFileTypeOther
		}

		if articleNotFound {
			entry.Streamable = false
			entry.Errors = append(entry.Errors, NZBContentFileErrorArticleNotFound)
		} else if fr.startErr != nil {
			entry.Streamable = false
			inspectLog.Warn("failed to fetch first segment", "error", fr.startErr, "name", filename)
			entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
		} else if fr.endErr != nil {
			entry.Streamable = false
			inspectLog.Warn("failed to fetch last segment", "error", fr.endErr, "name", filename)
			entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
		}

		if !entry.Streamable {
			content.Files = append(content.Files, entry)
			continue
		}

		if isArchive {
			af := &nzbArchiveFile{
				filetype: detectedType,
				name:     filename,
				size:     fr.nzbFile.Size(),
				volume:   -1,
			}
			if detectedType == FileTypeRAR && fr.startSegment != nil {
				boundaryBytes := append([]byte{}, fr.startSegment.Body...)
				if fr.endSegment != nil {
					boundaryBytes = append(boundaryBytes, fr.endSegment.Body...)
				}
				if vi, err := rardecode.ReadVolumeInfo(bytes.NewReader(boundaryBytes), rardecode.SkipCheck, rardecode.IterHeadersOnly); err == nil {
					af.volume = vi.Number
				}
			}
			nzbArchiveFiles = append(nzbArchiveFiles, af)
			continue
		}

		content.Files = append(content.Files, entry)
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

		ufs := NewUsenetFS(ctx, &UsenetFSConfig{
			NZB:               nzbDoc,
			Pool:              p,
			SegmentBufferSize: util.ToBytes("1MB"),
		})

		archiveName := name
		if group.Aliased {
			aliases := make(map[string]string, len(group.Files))
			for i, f := range group.Files {
				vol := group.Volumes[i]
				var syntheticName string
				switch group.FileType {
				case FileTypeRAR:
					syntheticName = GenerateRARVolumeName(group.BaseName, vol)
				case FileType7z:
					syntheticName = Generate7zVolumeName(group.BaseName, vol)
				}
				aliases[syntheticName] = f.Name()
				if vol == 0 {
					archiveName = syntheticName
					entry.Alias = syntheticName
				}
				entry.Parts = append(entry.Parts, NZBContentFile{
					Type:       NZBContentFileTypeArchive,
					Name:       f.Name(),
					Alias:      syntheticName,
					Size:       f.Size(),
					Volume:     vol,
					Streamable: true,
				})
			}
			ufs.SetAliases(aliases)
		} else {
			partAliases := normalizeRARPartNames(group.Files)
			if partAliases != nil {
				ufs.SetAliases(partAliases)
			}
			for i, f := range group.Files {
				vol := group.Volumes[i]
				part := NZBContentFile{
					Type:       NZBContentFileTypeArchive,
					Name:       f.Name(),
					Size:       f.Size(),
					Volume:     group.Volumes[i],
					Streamable: true,
				}
				for normalized, original := range partAliases {
					if original == f.Name() {
						part.Alias = normalized
						if vol == 0 {
							archiveName = normalized
							entry.Alias = normalized
						}
						break
					}
				}
				entry.Parts = append(entry.Parts, part)
			}
		}

		var archive Archive
		switch group.FileType {
		case FileTypeRAR:
			archive = NewRARArchive(ufs, archiveName)
		case FileType7z:
			archive = NewSevenZipArchive(ufs.toAfero(), archiveName)
		}

		if err := archive.Open(password); err != nil {
			inspectLog.Warn("failed to open archive", "error", err, "name", name)
			if errors.Is(err, ErrArticleNotFound) {
				entry.Errors = append(entry.Errors, NZBContentFileErrorArticleNotFound)
			} else {
				entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
			}
			content.Files = append(content.Files, entry)
			ufs.Close()
			continue
		}

		if streamable, err := archive.IsStreamable(); err != nil {
			inspectLog.Warn("failed to check archive streamability", "error", err, "name", name)
			if errors.Is(err, rardecode.ErrArchiveEncrypted) {
				entry.Errors = append(entry.Errors, NZBContentFileErrorMissingPassword)
			} else {
				entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
			}
			content.Files = append(content.Files, entry)
			ufs.Close()
			continue
		} else {
			entry.Streamable = streamable
		}
		if entry.Streamable {
			files, err := archive.GetFiles()
			if err != nil {
				inspectLog.Warn("failed to get archive files", "name", name, "error", err)
				if errors.Is(err, ErrArticleNotFound) {
					entry.Errors = append(entry.Errors, NZBContentFileErrorArticleNotFound)
				} else {
					entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
				}
			} else {
				entry.Files = p.inspectArchiveFiles(files, password)
			}
		}

		archive.Close()
		ufs.Close()
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
		for i, f := range group.Files {
			entry.Parts = append(entry.Parts, NZBContentFile{
				Type:       classifyNZBContentFileType(f.Name()),
				Name:       f.Name(),
				Size:       f.Size(),
				Volume:     group.Volumes[i],
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
			if errors.Is(err, ErrArticleNotFound) {
				entry.Errors = append(entry.Errors, NZBContentFileErrorArticleNotFound)
			} else {
				entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
			}
			afs.Close()
			result = append(result, entry)
			continue
		}

		if streamable, err := innerArchive.IsStreamable(); err != nil {
			inspectLog.Warn("failed to check nested archive streamability", "error", err, "name", name)
			entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
			afs.Close()
			result = append(result, entry)
			continue
		} else {
			entry.Streamable = streamable
		}
		if entry.Streamable {
			if innerFiles, err := innerArchive.GetFiles(); err != nil {
				inspectLog.Warn("failed to get nested archive files", "error", err, "name", name)
				if errors.Is(err, ErrArticleNotFound) {
					entry.Errors = append(entry.Errors, NZBContentFileErrorArticleNotFound)
				} else {
					entry.Errors = append(entry.Errors, NZBContentFileErrorOpenFailed)
				}
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
