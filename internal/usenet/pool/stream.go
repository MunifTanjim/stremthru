package usenet_pool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/util"
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

	cacheCapacity := 5
	if fileSize := file.TotalSize(); fileSize > 0 {
		if segmentSize := fileSize / int64(file.SegmentCount()); segmentSize > 0 {
			cacheCapacity = int(config.CacheSize / segmentSize)
		}
	}
	cache := NewSegmentCache(cacheCapacity)
	p.Log.Trace("segment cache created", "capacity", cacheCapacity)

	firstSegment, err := p.fetchFirstSegment(ctx, file, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file header: %w", err)
	}

	fileBytes := firstSegment.Body
	fileSize := firstSegment.Header.FileSize
	filename := file.GetName()
	fileType := DetectFileType(fileBytes, filename)

	p.Log.Trace("file type detected", "type", fileType, "filename", filename)

	switch fileType {
	case FileTypePlain:
		return p.streamPlainFile(ctx, file, fileSize, config, cache)
	case FileTypeRar:
		return p.streamRARFile(ctx, nzbDoc, config, cache)
	case FileType7z:
		return p.stream7zFile(ctx, nzbDoc, file, config)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func (p *Pool) fetchFirstSegment(
	ctx context.Context,
	file *nzb.File,
	cache *SegmentCache,
) (*decodedData, error) {

	p.Log.Trace("fetch first segment - start")

	firstSegment := &file.Segments[0]
	data, err := p.fetchSegment(ctx, firstSegment, file.Groups, cache)
	if err != nil {
		return nil, err
	}

	p.Log.Trace("fetch first segment - done", "size", data.Header.FileSize)

	return data, nil
}

func (p *Pool) streamPlainFile(
	ctx context.Context,
	file *nzb.File,
	fileSize int64,
	config *StreamConfig,
	cache *SegmentCache,
) (*Stream, error) {
	filename := file.GetName()

	p.Log.Trace("creating stream", "stream_type", "plain", "filename", filename, "file_size", fileSize, "segment_count", file.SegmentCount())

	// If cache wasn't created earlier (shouldn't happen), create it now
	if cache == nil && config.CacheSize > 0 && file.SegmentCount() > 0 {
		approxSegmentSize := fileSize / int64(file.SegmentCount())
		if approxSegmentSize > 0 {
			cacheCapacity := int(config.CacheSize / approxSegmentSize)
			cache = NewSegmentCache(cacheCapacity)
			p.Log.Trace("segment cache created (fallback)", "capacity", cacheCapacity, "approx_segment_size", approxSegmentSize)
		}
	}

	stream := NewFileStream(
		ctx,
		p,
		file,
		fileSize,
		config.SegmentBuffer,
		cache,
	)

	return &Stream{
		ReadSeekCloser: stream,
		Name:           filename,
		Size:           fileSize,
		ContentType:    GetContentType(filename),
	}, nil
}

func (p *Pool) streamRARFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	config *StreamConfig,
	cache *SegmentCache,
) (*Stream, error) {
	usenetFS := NewUsenetFS(ctx, &UsenetFSConfig{
		NZB:           nzbDoc,
		Pool:          p,
		SegmentBuffer: config.SegmentBuffer,
		Cache:         cache,
	})

	usenetRar := NewUsenetRARArchive(ctx, usenetFS)
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
	// // Fallback to compressed stream - collect RAR volume filenames
	// rarVolumes := make([]string, 0, len(rarParts))
	// for _, part := range rarParts {
	// 	rarVolumes = append(rarVolumes, part.Filename)
	// }
	// stream := NewCompressedRARFileStream(
	// 	ctx, usenetFS, rarVolumes,
	// 	targetVideo.Name, targetVideo.UnPackedSize,
	// 	config.Password,
	// )
	//
	// return &Stream{
	// 	ReadSeekCloser: stream,
	// 	Name:           targetVideo.Name,
	// 	Size:           targetVideo.UnPackedSize,
	// 	ContentType:    GetContentType(targetVideo.Name),
	// }, nil
}

// stream7zFile creates a stream for a file inside a 7z archive
func (p *Pool) stream7zFile(
	ctx context.Context,
	nzbDoc *nzb.NZB,
	archiveFile *nzb.File,
	config *StreamConfig,
) (*Stream, error) {
	// Collect all 7z parts from the NZB
	sevenZParts := p.collect7zParts(nzbDoc)
	if len(sevenZParts) == 0 {
		return nil, errors.New("no 7z archive parts found in NZB")
	}

	// For 7z, we need to download the entire archive to parse headers
	// (7z format requires random access for header parsing)
	// This is less efficient but necessary for the 7z format

	// Build archive parts info
	archiveParts := make([]ArchivePart, 0, len(sevenZParts))
	var totalSize int64
	for _, part := range sevenZParts {
		archiveParts = append(archiveParts, ArchivePart{
			Segments: part.Segments,
			Groups:   part.Groups,
			Size:     part.PartSize,
		})
		totalSize += part.PartSize
	}

	p.Log.Trace("stream7zFile collected parts", "part_count", len(sevenZParts), "total_size", totalSize)

	// Download archive data for header parsing
	archiveData := make([]byte, 0, totalSize)
	for _, part := range archiveParts {
		stream := NewFilePartStream(ctx, p, part.Segments, part.Groups, 5, nil)
		partData, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to download 7z part: %w", err)
		}
		archiveData = append(archiveData, partData...)
	}

	// Parse 7z headers
	readerAt := bytes.NewReader(archiveData)
	archiveInfo, err := Parse7zHeaders(readerAt, int64(len(archiveData)), config.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 7z headers: %w", err)
	}

	// Find the largest video file
	videos := FindVideoFilesIn7z(archiveInfo)
	if len(videos) == 0 {
		return nil, errors.New("no video files found in 7z archive")
	}

	var targetVideo SevenZFileEntry
	for _, v := range videos {
		if v.UnpackedSize > targetVideo.UnpackedSize {
			targetVideo = v
		}
	}

	p.Log.Trace("stream7zFile target selected", "filename", targetVideo.Name, "size", targetVideo.UnpackedSize)

	// 7z always requires extraction (no direct streaming support)
	stream := NewSevenZExtractStream(
		ctx, p, archiveParts,
		targetVideo.Name, targetVideo.UnpackedSize,
		config.Password,
	)

	return &Stream{
		ReadSeekCloser: stream,
		Name:           targetVideo.Name,
		Size:           targetVideo.UnpackedSize,
		ContentType:    GetContentType(targetVideo.Name),
	}, nil
}

// collect7zParts finds all 7z archive parts in the NZB and orders them
func (p *Pool) collect7zParts(nzbDoc *nzb.NZB) []SevenZPartInfo {
	var parts []SevenZPartInfo

	for i := range nzbDoc.Files {
		file := &nzbDoc.Files[i]
		filename := file.GetName()
		partNum := Get7zPartNumber(filename)

		if partNum >= 0 {
			parts = append(parts, SevenZPartInfo{
				PartNumber: partNum,
				PartSize:   file.TotalSize(),
				Segments:   file.Segments,
				Groups:     file.Groups,
			})
		}
	}

	return Group7zParts(parts)
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

	// Calculate total size from segments
	var totalSize int64
	for _, seg := range config.Segments {
		totalSize += seg.Bytes
	}

	// Set default buffer
	bufferSize := config.Buffer
	if bufferSize <= 0 {
		bufferSize = 5
	}

	stream := NewFilePartStream(ctx, p, config.Segments, config.Groups, bufferSize, nil)

	return &StreamSegmentsResult{
		ReadCloser: stream,
		Size:       totalSize,
	}, nil
}
