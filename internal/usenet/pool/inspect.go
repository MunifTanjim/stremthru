package usenet_pool

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"maps"
	"path/filepath"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/usenet/par2"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
	"github.com/nwaples/rardecode/v2"
)

var inspectLog = logger.Scoped("usenet/pool/inspect")

type NZBContentFileType string

const (
	NZBContentFileTypeVideo   NZBContentFileType = "video"
	NZBContentFileTypeArchive NZBContentFileType = "archive"
	NZBContentFileTypeParity  NZBContentFileType = "parity"
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

func computeMD5_16k(data []byte) [16]byte {
	if len(data) > 16384 {
		data = data[:16384]
	}
	return md5.Sum(data)
}

type nzbArchiveFile struct {
	entry    *NZBContentFile
	filetype FileType
	volume   int
}

func (f *nzbArchiveFile) Name() string {
	return f.entry.Name
}

func (f *nzbArchiveFile) Size() int64 {
	return f.entry.Size
}

func (f *nzbArchiveFile) FileType() FileType {
	return f.filetype
}

func (f *nzbArchiveFile) Volume() int {
	if f.volume >= 0 {
		return f.volume
	}
	name := getEffectiveName(f)
	switch f.filetype {
	case FileTypeRAR:
		return GetRARVolumeNumber(name)
	case FileType7z:
		return Get7zVolumeNumber(name)
	default:
		return -1
	}
}

func (f *nzbArchiveFile) Alias() string {
	return f.entry.Alias
}

type nzbParityFile struct {
	entry   *NZBContentFile
	nzbFile *nzb.File
}

func (p *Pool) InspectNZBContent(ctx context.Context, nzbDoc *nzb.NZB, password string) (*NZBContent, error) {
	content := &NZBContent{
		Files:      []NZBContentFile{},
		Streamable: true,
	}

	if len(nzbDoc.Files) == 0 {
		return content, nil
	}

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
	maxFetcher := max(
		p.maxPrimaryProviderConnections()-2*config.Newz.MaxConnectionPerStream,
		config.Newz.MaxConnectionPerStream,
	)
	fetchPool := pond.NewPool(maxFetcher)
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

	type par2MatchCandidate struct {
		entry        *NZBContentFile
		archiveFile  *nzbArchiveFile
		nzbFile      *nzb.File
		startSegment *SegmentData
	}

	var nzbArchiveFiles []*nzbArchiveFile
	var nzbPAR2Files []*nzbParityFile
	var par2Candidates []par2MatchCandidate

	for _, fr := range fetchResults {
		filename := fr.nzbFile.Name()

		articleNotFound := errors.Is(fr.startErr, ErrArticleNotFound) || errors.Is(fr.endErr, ErrArticleNotFound)

		var firstBytes []byte
		if fr.startSegment != nil {
			firstBytes = fr.startSegment.Body
		}
		detectedType := DetectFileType(firstBytes, filename)

		isParity := detectedType == FileTypePAR2
		isArchive := detectedType == FileTypeRAR || detectedType == FileType7z
		isVideoByExt := isVideoFile(filename)

		entry := &NZBContentFile{
			Name:       filename,
			Size:       fr.nzbFile.Size(),
			Streamable: true,
		}
		if isArchive {
			entry.Type = NZBContentFileTypeArchive
		} else if isVideoByExt {
			entry.Type = NZBContentFileTypeVideo
		} else if isParity {
			entry.Type = NZBContentFileTypeParity
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

		if detectedType == FileTypePAR2 {
			nzbPAR2Files = append(nzbPAR2Files, &nzbParityFile{
				entry:   entry,
				nzbFile: fr.nzbFile,
			})
			continue
		}

		if isArchive && entry.Streamable {
			af := &nzbArchiveFile{
				entry:    entry,
				filetype: detectedType,
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
			par2Candidates = append(par2Candidates, par2MatchCandidate{
				archiveFile:  af,
				nzbFile:      fr.nzbFile,
				startSegment: fr.startSegment,
			})
			continue
		}

		par2Candidates = append(par2Candidates, par2MatchCandidate{
			entry:        entry,
			nzbFile:      fr.nzbFile,
			startSegment: fr.startSegment,
		})
	}

	if len(nzbPAR2Files) > 0 && len(par2Candidates) > 0 {
		smallest := nzbPAR2Files[0].nzbFile
		for _, pa := range nzbPAR2Files[1:] {
			if pa.nzbFile.Size() < smallest.Size() {
				smallest = pa.nzbFile
			}
		}

		inspectLog.Debug("par2 index file found", "name", smallest.Name(), "size", smallest.Size(), "segments", smallest.SegmentCount())

		par2Stream, err := NewFileStream(ctx, p, smallest, smallest.Size())
		if err != nil {
			inspectLog.Warn("failed to create par2 stream", "error", err, "name", smallest.Name())
		} else {
			decoder := par2.NewDecoder(par2Stream)
			par2File, err := decoder.Decode()
			if err != nil {
				inspectLog.Warn("failed to decode par2 file", "error", err, "name", smallest.Name())
			} else {
				par2Index := make(map[[16]byte]*par2.FileDescriptionPacket, len(par2File.Files))
				for i := range par2File.Files {
					fd := &par2File.Files[i]
					par2Index[fd.MD5_16k] = fd
					inspectLog.Trace("par2 file entry", "filename", fd.Filename, "size", fd.Length, "md5_16k", fmt.Sprintf("%x", fd.MD5_16k))
				}

				inspectLog.Debug("par2 index built", "entries", len(par2Index))

				for i := range par2Candidates {
					c := &par2Candidates[i]
					if c.startSegment == nil || len(c.startSegment.Body) == 0 {
						continue
					}
					if c.startSegment.FileSize >= 16384 && len(c.startSegment.Body) < 16384 {
						inspectLog.Trace("par2 match skipped: first segment too short for MD5_16k", "name", c.nzbFile.Name(), "body_len", len(c.startSegment.Body))
						continue
					}
					hash := computeMD5_16k(c.startSegment.Body)
					fd, matched := par2Index[hash]
					inspectLog.Trace("candidate", "name", c.nzbFile.Name(), "md5_16k", fmt.Sprintf("%x", hash), "matched", matched)
					if !matched {
						continue
					}
					inspectLog.Trace("par2 matched file", "original_name", c.nzbFile.Name(), "par2_name", fd.Filename)

					entry := c.entry
					if c.archiveFile != nil {
						entry = c.archiveFile.entry
					}
					entry.Alias = fd.Filename
					if entry.Type == NZBContentFileTypeOther {
						entry.Type = classifyNZBContentFileType(fd.Filename)
					}
					if entry.Type == NZBContentFileTypeArchive {
						if c.archiveFile == nil {
							c.archiveFile = &nzbArchiveFile{
								entry:    entry,
								filetype: DetectArchiveFileTypeByExtension(fd.Filename),
								volume:   -1,
							}
						}
					}
				}
			}
			par2Stream.Close()
		}
	}

	for i := range par2Candidates {
		c := &par2Candidates[i]
		if c.archiveFile != nil {
			nzbArchiveFiles = append(nzbArchiveFiles, c.archiveFile)
		} else if c.entry != nil {
			content.Files = append(content.Files, *c.entry)
		}
	}

	for _, pf := range nzbPAR2Files {
		content.Files = append(content.Files, *pf.entry)
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
			oldRARNaming := group.FileType == FileTypeRAR && !HasRARNewVolumeName(group.Files[0].Name())
			for i, f := range group.Files {
				vol := group.Volumes[i]
				var syntheticName string
				switch group.FileType {
				case FileTypeRAR:
					syntheticName = GenerateRARVolumeName(group.BaseName, vol, oldRARNaming)
				case FileType7z:
					syntheticName = Generate7zVolumeName(group.BaseName, vol)
				}
				aliases[syntheticName] = f.Name()
				if isFirstVolume(group.FileType, vol) {
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
			aliases := map[string]string{}
			if group.FileType == FileTypeRAR {
				partAliases := normalizeRARPartNames(group.Files)
				maps.Copy(aliases, partAliases)
			}
			for i, f := range group.Files {
				if alias := f.Alias(); alias != "" {
					aliases[alias] = f.Name()
				}
				vol := group.Volumes[i]
				part := NZBContentFile{
					Type:       NZBContentFileTypeArchive,
					Name:       f.Name(),
					Size:       f.Size(),
					Volume:     group.Volumes[i],
					Streamable: true,
				}
				for alias, name := range aliases {
					if name == f.Name() {
						part.Alias = alias
						if isFirstVolume(group.FileType, vol) {
							archiveName = alias
							entry.Alias = alias
						}
						break
					}
				}
				entry.Parts = append(entry.Parts, part)
			}
			ufs.SetAliases(aliases)
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
