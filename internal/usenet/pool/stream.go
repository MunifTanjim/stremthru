package usenet_pool

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var (
	_ io.ReadSeekCloser = (*Stream)(nil)
)

type StreamConfig struct {
	Password          string
	SegmentBufferSize int64
}

type Stream struct {
	io.ReadSeekCloser
	Name        string
	Size        int64
	ContentType string
}

func (p *Pool) streamFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	fileIdx int,
	config *StreamConfig,
) (*Stream, error) {
	if config == nil {
		config = &StreamConfig{}
	}

	if fileIdx < 0 || fileIdx >= nzbDoc.FileCount() {
		return nil, fmt.Errorf("file index %d out of range [0, %d)", fileIdx, nzbDoc.FileCount())
	}

	file := &nzbDoc.Files[fileIdx]
	if file.SegmentCount() == 0 {
		return nil, errors.New("file has no segments")
	}

	p.Log.Trace("found file", "idx", fileIdx, "name", file.Name(), "segment_count", file.SegmentCount())

	firstSegment, err := p.fetchFirstSegment(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file header: %w", err)
	}

	fileBytes := firstSegment.Body
	filename := file.Name()
	fileType := DetectFileType(fileBytes, filename)

	p.Log.Trace("file type detected", "type", fileType, "filename", filename)

	switch fileType {
	case FileTypePlain:
		return p.streamPlainFile(file, config)
	case FileTypeRAR:
		return p.streamRARFile(ctx, nzbDoc, config)
	case FileType7z:
		return p.stream7zFile(ctx, nzbDoc, config)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func (p *Pool) fetchFirstSegment(
	ctx context.Context,
	file *nzb.File,
) (*SegmentData, error) {
	p.Log.Trace("fetch first segment - start")

	firstSegment := &file.Segments[0]
	data, err := p.fetchSegment(ctx, firstSegment, file.Groups)
	if err != nil {
		return nil, err
	}

	p.Log.Trace("fetch first segment - done", "size", data.Size)

	return data, nil
}

func (p *Pool) streamPlainFile(
	file *nzb.File,
	config *StreamConfig,
) (*Stream, error) {
	filename := file.Name()

	p.Log.Trace("creating stream", "stream_type", "plain", "filename", filename, "segment_count", file.SegmentCount())

	stream, err := NewFileStream(
		p,
		file,
		config.SegmentBufferSize,
	)
	if err != nil {
		return nil, err
	}

	return &Stream{
		ReadSeekCloser: stream,
		Name:           filename,
		Size:           stream.Size(),
		ContentType:    GetContentType(filename),
	}, nil
}

func filterVideoFiles(files []ArchiveFile) []ArchiveFile {
	videos := make([]ArchiveFile, 0)
	for _, f := range files {
		if isVideoFile(f.Name()) {
			videos = append(videos, f)
		}
	}
	return videos
}

func (p *Pool) streamArchiveFile(
	archive Archive,
	archiveType FileType,
) (*Stream, error) {
	if !archive.IsStreamable() {
		return nil, fmt.Errorf("non-streamable %s archive", archiveType)
	}

	files, err := archive.GetFiles()
	if err != nil {
		return nil, err
	}

	if archiveGroups := groupArchiveVolumes(files); len(archiveGroups) > 0 {
		p.Log.Trace("stream archive file - found nested archives, trying them first", "type", archiveType)
		stream, err := p.streamNestedArchive(archiveGroups)
		if err == nil {
			return stream, nil
		}
		p.Log.Debug("stream archive file - nested archive failed, falling back to direct video", "error", err)
	}

	videos := filterVideoFiles(files)
	if len(videos) == 0 {
		return nil, fmt.Errorf("no video files or nested archives found in %s archive", archiveType)
	}

	return p.streamVideoFromArchive(videos, archiveType)
}

func (p *Pool) streamVideoFromArchive(videos []ArchiveFile, archiveType FileType) (*Stream, error) {
	file := slices.MaxFunc(videos, func(a, b ArchiveFile) int {
		return cmp.Compare(a.Size(), b.Size())
	})

	p.Log.Trace("stream archive file - target selected", "type", archiveType, "filename", file.Name())

	if !file.IsStreamable() {
		return nil, fmt.Errorf("non-streamable file in %s archive", archiveType)
	}

	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open: %w", err)
	}

	return &Stream{
		ReadSeekCloser: r,
		Name:           file.Name(),
		Size:           file.Size(),
		ContentType:    GetContentType(file.Name()),
	}, nil
}

func (p *Pool) streamNestedArchive(archiveGroups []archiveVolumeGroup[ArchiveFile]) (*Stream, error) {
	var lastErr error
	for i := range archiveGroups {
		group := &archiveGroups[i]
		p.Log.Trace("stream nested archive - trying group",
			"base_name", group.BaseName,
			"type", group.FileType,
			"parts", len(group.Files),
			"total_size", group.TotalSize)

		stream, err := p.tryStreamNestedArchiveGroup(group)
		if err != nil {
			p.Log.Debug("stream nested archive - group failed", "error", err)
			lastErr = err
			continue
		}
		return stream, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to stream nested archive: %w", lastErr)
	}
	return nil, fmt.Errorf("no streamable content found in nested archives")
}

func (p *Pool) tryStreamNestedArchiveGroup(group *archiveVolumeGroup[ArchiveFile]) (*Stream, error) {
	for _, f := range group.Files {
		if !f.IsStreamable() {
			return nil, fmt.Errorf("inner archive part %s is not streamable", f.Name())
		}
	}

	afs := NewArchiveFS(group.Files)

	var innerArchive Archive
	switch group.FileType {
	case FileTypeRAR:
		innerArchive = NewRARArchive(afs, filepath.Base(group.Files[0].Name()))
	case FileType7z:
		innerArchive = NewSevenZipArchive(afs.toAfero(), filepath.Base(group.Files[0].Name()))
	default:
		afs.Close()
		return nil, fmt.Errorf("unsupported inner archive type: %s", group.FileType)
	}

	if err := innerArchive.Open(""); err != nil {
		afs.Close()
		return nil, fmt.Errorf("failed to open inner archive: %w", err)
	}

	stream, err := p.streamArchiveFileInner(innerArchive, group.FileType)
	if err != nil {
		innerArchive.Close()
		return nil, err
	}

	return &Stream{
		ReadSeekCloser: &nestedArchiveStream{
			ReadSeekCloser: stream.ReadSeekCloser,
			innerArchive:   innerArchive,
		},
		Name:        stream.Name,
		Size:        stream.Size,
		ContentType: stream.ContentType,
	}, nil
}

func (p *Pool) streamArchiveFileInner(archive Archive, archiveType FileType) (*Stream, error) {
	if !archive.IsStreamable() {
		return nil, fmt.Errorf("non-streamable inner %s archive", archiveType)
	}

	files, err := archive.GetFiles()
	if err != nil {
		return nil, err
	}

	videos := filterVideoFiles(files)
	if len(videos) == 0 {
		return nil, fmt.Errorf("no video files found in inner %s archive", archiveType)
	}

	return p.streamVideoFromArchive(videos, archiveType)
}

type nestedArchiveStream struct {
	io.ReadSeekCloser
	innerArchive Archive
}

func (nas *nestedArchiveStream) Close() error {
	streamErr := nas.ReadSeekCloser.Close()
	archiveErr := nas.innerArchive.Close()
	return errors.Join(streamErr, archiveErr)
}

func (p *Pool) streamRARFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
) (*Stream, error) {
	ufs := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:               nzbDoc,
		Pool:              p,
		SegmentBufferSize: config.SegmentBufferSize,
	})
	archive := NewUsenetRARArchive(ufs)
	if err := archive.Open(config.Password); err != nil {
		return nil, err
	}
	return p.streamArchiveFile(archive, FileTypeRAR)
}

func (p *Pool) stream7zFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
) (*Stream, error) {
	ufs := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:               nzbDoc,
		Pool:              p,
		SegmentBufferSize: config.SegmentBufferSize,
	})
	archive := NewUsenetSevenZipArchive(ufs)
	if err := archive.Open(config.Password); err != nil {
		return nil, err
	}
	return p.streamArchiveFile(archive, FileType7z)
}

func (p *Pool) StreamLargestFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
) (*Stream, error) {
	if len(nzbDoc.Files) == 0 {
		return nil, errors.New("NZB has no files")
	}

	largestFileIdx := nzbDoc.GetLargestFileIdx(func(filename string) bool {
		return !isVideoFile(filename) && !IsArchiveFile(filename)
	})

	p.Log.Trace("found largest file", "idx", largestFileIdx)

	return p.streamFile(ctx, nzbDoc, largestFileIdx, config)
}

func (p *Pool) StreamFileByName(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	filename string,
	config *StreamConfig,
) (*Stream, error) {
	for i := range nzbDoc.Files {
		if strings.EqualFold(nzbDoc.Files[i].Name(), filename) {
			return p.streamFile(ctx, nzbDoc, i, config)
		}
	}
	return nil, fmt.Errorf("no file matching '%s' found", filename)
}

func (p *Pool) streamTargetFromArchive(
	archive Archive,
	targetParts []string,
	archiveType FileType,
) (*Stream, error) {
	files, err := archive.GetFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive files: %w", err)
	}

	targetName := strings.Trim(targetParts[0], "/")
	remainingParts := targetParts[1:]

	for _, f := range files {
		if !strings.EqualFold(f.Name(), targetName) {
			continue
		}

		if len(remainingParts) == 0 {
			if !f.IsStreamable() {
				return nil, fmt.Errorf("file %s is not streamable", f.Name())
			}
			r, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", f.Name(), err)
			}
			return &Stream{
				ReadSeekCloser: r,
				Name:           f.Name(),
				Size:           f.Size(),
				ContentType:    GetContentType(f.Name()),
			}, nil
		}

		if !f.IsStreamable() {
			return nil, fmt.Errorf("inner archive %s is not streamable", f.Name())
		}

		innerFileType := DetectArchiveFileTypeByExtension(f.Name())

		archiveGroups := groupArchiveVolumes(files)
		var matchedGroup *archiveVolumeGroup[ArchiveFile]
		for i := range archiveGroups {
			for _, gf := range archiveGroups[i].Files {
				if strings.EqualFold(gf.Name(), targetName) {
					matchedGroup = &archiveGroups[i]
					break
				}
			}
			if matchedGroup != nil {
				break
			}
		}

		archiveFiles := []ArchiveFile{f}
		archiveFileType := innerFileType
		if matchedGroup != nil {
			for _, mf := range matchedGroup.Files {
				if !mf.IsStreamable() {
					return nil, fmt.Errorf("inner archive part %s is not streamable", mf.Name())
				}
			}
			archiveFiles = matchedGroup.Files
			archiveFileType = matchedGroup.FileType
		}

		var innerArchive Archive
		afs := NewArchiveFS(archiveFiles)
		switch archiveFileType {
		case FileTypeRAR:
			innerArchive = NewRARArchive(afs, filepath.Base(archiveFiles[0].Name()))
		case FileType7z:
			innerArchive = NewSevenZipArchive(afs.toAfero(), filepath.Base(archiveFiles[0].Name()))
		default:
			afs.Close()
			return nil, fmt.Errorf("unsupported inner archive type: %s", archiveFileType)
		}
		innerFileType = archiveFileType

		if err := innerArchive.Open(""); err != nil {
			innerArchive.Close()
			return nil, fmt.Errorf("failed to open inner archive: %w", err)
		}

		if !innerArchive.IsStreamable() {
			innerArchive.Close()
			return nil, fmt.Errorf("non-streamable inner %s archive", innerFileType)
		}

		stream, err := p.streamTargetFromArchive(innerArchive, remainingParts, innerFileType)
		if err != nil {
			innerArchive.Close()
			return nil, err
		}

		return &Stream{
			ReadSeekCloser: &nestedArchiveStream{
				ReadSeekCloser: stream.ReadSeekCloser,
				innerArchive:   innerArchive,
			},
			Name:        stream.Name,
			Size:        stream.Size,
			ContentType: stream.ContentType,
		}, nil
	}

	return nil, fmt.Errorf("no file matching '%s' found in archive", targetName)
}

func (p *Pool) StreamByContentPath(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	contentPath []string,
	config *StreamConfig,
) (*Stream, error) {
	if len(contentPath) == 0 || contentPath[0] == "" {
		return nil, fmt.Errorf("invalid content path: %s", strings.Join(contentPath, "::"))
	}

	if config == nil {
		config = &StreamConfig{}
	}

	if len(contentPath) == 1 {
		filename := strings.Trim(contentPath[0], "/")
		for i := range nzbDoc.Files {
			if strings.EqualFold(nzbDoc.Files[i].Name(), filename) {
				return p.streamPlainFile(&nzbDoc.Files[i], config)
			}
		}
		return nil, fmt.Errorf("no file matching '%s' found", filename)
	}

	archiveName := strings.Trim(contentPath[0], "/")
	var archiveFile *nzb.File
	for i := range nzbDoc.Files {
		if strings.EqualFold(nzbDoc.Files[i].Name(), archiveName) {
			archiveFile = &nzbDoc.Files[i]
			break
		}
	}
	if archiveFile == nil {
		return nil, fmt.Errorf("no archive file matching '%s' found", archiveName)
	}

	firstSegment, err := p.fetchFirstSegment(ctx, archiveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch archive header: %w", err)
	}
	fileType := DetectFileType(firstSegment.Body, archiveFile.Name())

	ufs := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:               nzbDoc,
		Pool:              p,
		SegmentBufferSize: config.SegmentBufferSize,
	})

	var archive Archive
	switch fileType {
	case FileTypeRAR:
		archive = NewUsenetRARArchive(ufs)
	case FileType7z:
		archive = NewUsenetSevenZipArchive(ufs)
	default:
		return nil, fmt.Errorf("file '%s' is not an archive", archiveName)
	}

	if err := archive.Open(config.Password); err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}

	if !archive.IsStreamable() {
		archive.Close()
		return nil, fmt.Errorf("non-streamable %s archive", fileType)
	}

	stream, err := p.streamTargetFromArchive(archive, contentPath[1:], fileType)
	if err != nil {
		archive.Close()
		return nil, err
	}

	return stream, nil
}

type StreamSegmentsConfig struct {
	Segments   []nzb.Segment // Segments to stream
	Groups     []string      // Newsgroups
	BufferSize int64
}

type StreamSegmentsResult struct {
	io.ReadCloser
	Size int64
}

func (p *Pool) StreamSegments(
	ctx context.Context,
	conf StreamSegmentsConfig,
) (*StreamSegmentsResult, error) {
	if len(conf.Segments) == 0 {
		return nil, errors.New("no segments provided")
	}

	f := &nzb.File{
		Segments: conf.Segments,
		Groups:   conf.Groups,
	}
	firstSegment, err := p.fetchFirstSegment(ctx, f)
	if err != nil {
		return nil, err
	}

	stream, err := NewFileStream(p, f, conf.BufferSize)
	if err != nil {
		return nil, err
	}

	return &StreamSegmentsResult{
		ReadCloser: stream,
		Size:       firstSegment.FileSize,
	}, nil
}
