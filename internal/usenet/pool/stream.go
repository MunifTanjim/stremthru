package usenet_pool

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
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

	p.Log.Trace("found file", "idx", fileIdx, "name", file.GetName(), "segment_count", file.SegmentCount())

	firstSegment, err := p.fetchFirstSegment(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file header: %w", err)
	}

	fileBytes := firstSegment.Body
	filename := file.GetName()
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
	filename := file.GetName()

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
		return cmp.Compare(a.UnPackedSize(), b.UnPackedSize())
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
		Size:           file.UnPackedSize(),
		ContentType:    GetContentType(file.Name()),
	}, nil
}

func (p *Pool) streamNestedArchive(archiveGroups []archiveVolumeGroup) (*Stream, error) {
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

func (p *Pool) tryStreamNestedArchiveGroup(group *archiveVolumeGroup) (*Stream, error) {
	for _, f := range group.Files {
		if !f.IsStreamable() {
			return nil, fmt.Errorf("inner archive part %s is not streamable", f.Name())
		}
	}

	afs := NewArchiveFS(group.Files)

	var innerArchive Archive
	switch group.FileType {
	case FileTypeRAR:
		innerArchive = NewRARArchive(afs, group.GetFirstVolumeName())
	case FileType7z:
		innerArchive = NewSevenZipArchive(afs.toAfero(), group.GetFirstVolumeName())
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

	largestFileIdx := nzbDoc.GetLargestFileIdx()

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
		if strings.EqualFold(nzbDoc.Files[i].GetName(), filename) {
			return p.streamFile(ctx, nzbDoc, i, config)
		}
	}
	return nil, fmt.Errorf("no file matching '%s' found", filename)
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
