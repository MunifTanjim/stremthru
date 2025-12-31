package usenet_pool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

type ArchivePart struct {
	Segments []nzb.Segment
	Groups   []string
	Size     int64
}

// ArchiveFileLocation describes where a file's data is within archive parts
type ArchiveFileLocation struct {
	Name      string
	Size      int64
	Parts     []ArchiveFilePart
	Encrypted bool
}

// ArchiveFilePart describes a portion of a file within an archive part
type ArchiveFilePart struct {
	PartIndex       int       // Which archive part
	ByteRangeInPart ByteRange // Where in the archive part
	ByteRangeInFile ByteRange // What portion of the file
}

// limitedMultiSegmentStream wraps MultiSegmentStream with a read limit
type limitedMultiSegmentStream struct {
	stream    *FilePartStream
	remaining int64
}

func (l *limitedMultiSegmentStream) Read(p []byte) (n int, err error) {
	if l.remaining <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err = l.stream.Read(p)
	l.remaining -= int64(n)
	return n, err
}

func (l *limitedMultiSegmentStream) Close() error {
	return l.stream.Close()
}

// CompressedRARFileStream streams a file from RAR archive using decompression
// This is used when the file is compressed (not store mode)
type CompressedRARFileStream struct {
	usenetFS   *UsenetFS
	rarVolumes []string // ordered RAR volume filenames
	filename   string
	password   string
	fileSize   int64

	mu       sync.Mutex
	position int64
	buffer   *bytes.Buffer // Buffer for decompressed data
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewCompressedRarExtractStream creates a stream that extracts from RAR via decompression
func NewCompressedRARFileStream(
	ctx context.Context,
	usenetFS *UsenetFS,
	rarVolumes []string,
	filename string,
	fileSize int64,
	password string,
) *CompressedRARFileStream {
	ctx, cancel := context.WithCancel(ctx)
	return &CompressedRARFileStream{
		usenetFS:   usenetFS,
		rarVolumes: rarVolumes,
		filename:   filename,
		fileSize:   fileSize,
		password:   password,
		buffer:     bytes.NewBuffer(nil),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Read implements io.Reader
func (s *CompressedRARFileStream) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.position >= s.fileSize {
		return 0, io.EOF
	}

	// If buffer is empty, extract more data
	if s.buffer.Len() == 0 {
		if err := s.extractToBuffer(); err != nil {
			return 0, err
		}
	}

	n, err = s.buffer.Read(p)
	s.position += int64(n)
	return n, err
}

// Seek implements io.Seeker
// Note: Seeking backwards requires re-extraction from the beginning
func (s *CompressedRARFileStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.position + offset
	case io.SeekEnd:
		newPos = s.fileSize + offset
	default:
		return s.position, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return s.position, fmt.Errorf("negative position: %d", newPos)
	}
	if newPos > s.fileSize {
		newPos = s.fileSize
	}

	if newPos < s.position {
		// Backward seek - need to re-extract from beginning
		s.buffer.Reset()
		s.position = 0
	}

	// Forward seek - discard data
	for s.position < newPos {
		remaining := newPos - s.position
		if s.buffer.Len() == 0 {
			if err := s.extractToBuffer(); err != nil {
				return s.position, err
			}
		}
		toDiscard := remaining
		if int64(s.buffer.Len()) < toDiscard {
			toDiscard = int64(s.buffer.Len())
		}
		s.buffer.Next(int(toDiscard))
		s.position += toDiscard
	}

	return s.position, nil
}

// Close implements io.Closer
func (s *CompressedRARFileStream) Close() error {
	s.cancel()
	s.buffer.Reset()
	return nil
}

// Size returns the total file size
func (s *CompressedRARFileStream) Size() int64 {
	return s.fileSize
}

// extractToBuffer extracts data from the RAR archive into the buffer
func (s *CompressedRARFileStream) extractToBuffer() error {
	// Open all RAR volume files from UsenetFS
	readers := make([]io.Reader, 0, len(s.rarVolumes))
	openedFiles := make([]io.Closer, 0, len(s.rarVolumes))
	defer func() {
		for _, f := range openedFiles {
			f.Close()
		}
	}()

	for _, volumeName := range s.rarVolumes {
		file, err := s.usenetFS.Open(volumeName)
		if err != nil {
			return fmt.Errorf("failed to open RAR volume %s: %w", volumeName, err)
		}
		openedFiles = append(openedFiles, file)
		readers = append(readers, file)
	}
	combined := io.MultiReader(readers...)

	// Open RAR archive
	opts := []rardecode.Option{}
	if s.password != "" {
		opts = append(opts, rardecode.Password(s.password))
	}

	reader, err := rardecode.NewReader(combined, opts...)
	if err != nil {
		return fmt.Errorf("failed to open RAR: %w", err)
	}

	// Find and extract the target file
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return fmt.Errorf("file %s not found in archive", s.filename)
		}
		if err != nil {
			return fmt.Errorf("failed to read RAR: %w", err)
		}

		if header.Name == s.filename {
			// Extract into buffer
			_, err := io.Copy(s.buffer, reader)
			if err != nil {
				return fmt.Errorf("failed to extract: %w", err)
			}
			return nil
		}
	}
}

// SevenZExtractStream streams a file from 7z archive using decompression
type SevenZExtractStream struct {
	pool     *Pool
	parts    []ArchivePart
	filename string
	password string
	fileSize int64

	mu       sync.Mutex
	position int64
	buffer   *bytes.Buffer
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSevenZExtractStream creates a stream that extracts from 7z
func NewSevenZExtractStream(
	ctx context.Context,
	pool *Pool,
	parts []ArchivePart,
	filename string,
	fileSize int64,
	password string,
) *SevenZExtractStream {
	ctx, cancel := context.WithCancel(ctx)
	return &SevenZExtractStream{
		pool:     pool,
		parts:    parts,
		filename: filename,
		fileSize: fileSize,
		password: password,
		buffer:   bytes.NewBuffer(nil),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Read implements io.Reader
func (s *SevenZExtractStream) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.position >= s.fileSize {
		return 0, io.EOF
	}

	if s.buffer.Len() == 0 {
		if err := s.extractToBuffer(); err != nil {
			return 0, err
		}
	}

	n, err = s.buffer.Read(p)
	s.position += int64(n)
	return n, err
}

// Seek implements io.Seeker
func (s *SevenZExtractStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.position + offset
	case io.SeekEnd:
		newPos = s.fileSize + offset
	default:
		return s.position, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return s.position, fmt.Errorf("negative position: %d", newPos)
	}
	if newPos > s.fileSize {
		newPos = s.fileSize
	}

	if newPos < s.position {
		s.buffer.Reset()
		s.position = 0
	}

	for s.position < newPos {
		remaining := newPos - s.position
		if s.buffer.Len() == 0 {
			if err := s.extractToBuffer(); err != nil {
				return s.position, err
			}
		}
		toDiscard := remaining
		if int64(s.buffer.Len()) < toDiscard {
			toDiscard = int64(s.buffer.Len())
		}
		s.buffer.Next(int(toDiscard))
		s.position += toDiscard
	}

	return s.position, nil
}

// Close implements io.Closer
func (s *SevenZExtractStream) Close() error {
	s.cancel()
	s.buffer.Reset()
	return nil
}

// Size returns the total file size
func (s *SevenZExtractStream) Size() int64 {
	return s.fileSize
}

// extractToBuffer extracts the 7z file into buffer
func (s *SevenZExtractStream) extractToBuffer() error {
	// Download all parts into memory for 7z (requires ReaderAt)
	var totalSize int64
	for _, part := range s.parts {
		totalSize += part.Size
	}

	data := make([]byte, 0, totalSize)
	for _, part := range s.parts {
		stream := NewFilePartStream(s.ctx, s.pool, part.Segments, part.Groups, 5, nil)
		partData, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			return fmt.Errorf("failed to download archive part: %w", err)
		}
		data = append(data, partData...)
	}

	// Create ReaderAt from data
	readerAt := bytes.NewReader(data)

	// Open 7z archive
	var reader *sevenzip.Reader
	var err error
	if s.password != "" {
		reader, err = sevenzip.NewReaderWithPassword(readerAt, int64(len(data)), s.password)
	} else {
		reader, err = sevenzip.NewReader(readerAt, int64(len(data)))
	}
	if err != nil {
		return fmt.Errorf("failed to open 7z: %w", err)
	}

	// Find and extract the target file
	for _, file := range reader.File {
		if file.Name == s.filename {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in 7z: %w", err)
			}
			defer rc.Close()

			_, err = io.Copy(s.buffer, rc)
			if err != nil {
				return fmt.Errorf("failed to extract: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("file %s not found in archive", s.filename)
}
