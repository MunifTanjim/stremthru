package usenet_pool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var (
	_ io.ReadSeekCloser = (*Stream)(nil)
)

type StreamConfig struct {
	CacheSize     int64
	Password      string
	SegmentBuffer int
}

func (c *StreamConfig) setDefaults() {
	if c.CacheSize <= 0 {
		c.CacheSize = util.ToBytes("50 MB")
	}
	if c.SegmentBuffer <= 0 {
		c.SegmentBuffer = 5
	}
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
	config.setDefaults()

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
	case FileTypeRar:
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

	p.Log.Trace("fetch first segment - done", "size", data.FileSize)

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
		config.SegmentBuffer,
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

func (p *Pool) streamRARFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
) (*Stream, error) {
	usenetFS := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:           nzbDoc,
		Pool:          p,
		SegmentBuffer: config.SegmentBuffer,
	})

	usenetRar := NewUsenetRARArchive(usenetFS)
	usenetRar.Password = config.Password

	if solid, err := usenetRar.IsSolid(); err != nil {
		return nil, fmt.Errorf("failed to check RAR solid flag: %w", err)
	} else if solid {
		return nil, errors.New("solid RAR archives are not supported for streaming")
	}

	videos := []UsenetRARFile{}

	files, err := usenetRar.GetFiles()
	if err != nil {
		return nil, err
	}
	for i := range files {
		f := &files[i]
		if isVideoFile(f.Name) {
			videos = append(videos, *f)
		}
	}
	if len(videos) == 0 {
		return nil, errors.New("no video files found in RAR archive")
	}

	targetVideo := &videos[0]
	for i := range videos[1:] {
		f := &videos[i+1]
		if f.UnPackedSize > targetVideo.UnPackedSize {
			targetVideo = f
		}
	}

	isStreamable := targetVideo.IsStreamable()
	p.Log.Trace("stream rar file - target selected", "filename", targetVideo.Name)

	if isStreamable {
		rarFS, err := usenetRar.OpenFS()
		if err != nil {
			return nil, fmt.Errorf("failed to open RarFS: %w", err)
		}

		rarFile, err := rarFS.Open(targetVideo.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to open file in RarFS: %w", err)
		}

		return &Stream{
			ReadSeekCloser: rarFile.(io.ReadSeekCloser),
			Name:           targetVideo.Name,
			Size:           targetVideo.UnPackedSize,
			ContentType:    GetContentType(targetVideo.Name),
		}, nil
	}

	return nil, errors.New("non-streamable RAR files are not supported in this implementation")
}

// stream7zFile creates a stream for a file inside a 7z archive
func (p *Pool) stream7zFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
) (*Stream, error) {
	ufs := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:           nzbDoc,
		Pool:          p,
		SegmentBuffer: config.SegmentBuffer,
	})

	usenet7z := NewUsenet7zArchive(ufs)
	err := usenet7z.Open(config.Password)
	if err != nil {
		return nil, err
	}

	videos := []Usenet7zFile{}

	files, err := usenet7z.GetFiles()
	if err != nil {
		return nil, err
	}
	for i := range files {
		f := &files[i]
		if isVideoFile(f.Name) {
			videos = append(videos, *f)
		}
	}
	if len(videos) == 0 {
		return nil, errors.New("no video files found in 7z archive")
	}

	targetVideo := &videos[0]
	println(targetVideo.Name)
	for i := range videos[1:] {
		f := &videos[i+1]
		println(f.Name)
		if f.UnPackedSize > targetVideo.UnPackedSize {
			targetVideo = f
		}
	}

	isStreamable := !targetVideo.IsCompressed()
	p.Log.Trace("stream 7z file - target selected", "filename", targetVideo.Name)

	if isStreamable {
		s, err := targetVideo.toStream()
		if err != nil {
			return nil, fmt.Errorf("failed to open: %w", err)
		}
		return s, nil
	}

	return nil, errors.New("non-streamable 7z files are not supported in this implementation")
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

var _ io.ReadSeekCloser = (*Stream)(nil)

// StreamSegmentsConfig configures direct segment streaming
type StreamSegmentsConfig struct {
	Segments []nzb.Segment // Segments to stream
	Groups   []string      // Newsgroups
	Buffer   int           // Prefetch buffer size (default: 5)
}

// StreamSegmentsResult contains the result of streaming segments
type StreamSegmentsResult struct {
	io.ReadCloser
	Size int64
}

// StreamSegments streams segments directly without NZB document
// This is a lower-level API for when you have raw segments
func (p *Pool) StreamSegments(
	ctx context.Context,
	config StreamSegmentsConfig,
) (*StreamSegmentsResult, error) {
	if len(config.Segments) == 0 {
		return nil, errors.New("no segments provided")
	}

	// Set default buffer
	bufferSize := config.Buffer
	if bufferSize <= 0 {
		bufferSize = 5
	}

	f := &nzb.File{
		Segments: config.Segments,
		Groups:   config.Groups,
	}
	firstSegment, err := p.fetchFirstSegment(ctx, f)
	if err != nil {
		return nil, err
	}

	stream, err := NewFileStream(p, f, bufferSize)
	if err != nil {
		return nil, err
	}

	return &StreamSegmentsResult{
		ReadCloser: stream,
		Size:       firstSegment.FileSize,
	}, nil
}
