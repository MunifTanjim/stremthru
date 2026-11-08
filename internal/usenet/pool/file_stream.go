package usenet_pool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

// [Start,End)
type ByteRange struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func (r ByteRange) Count() int64 {
	return r.End - r.Start
}

func (r ByteRange) Contains(byteIdx int64) bool {
	return r.Start <= byteIdx && byteIdx < r.End
}

func (r ByteRange) ContainsRange(other ByteRange) bool {
	return r.Start <= other.Start && other.End <= r.End
}

func NewByteRangeFromSize(start, size int64) ByteRange {
	return ByteRange{Start: start, End: start + size}
}

var (
	_ io.ReadSeekCloser = (*FileStream)(nil)
	_ io.ReaderAt       = (*FileStream)(nil)
)

var fileLog = logger.Scoped("usenet/pool/file_stream")

type FileStream struct {
	file             *nzb.File
	fileSize         int64
	avgSegmentSize   int64
	segmentSizeRatio float64

	pool       *Pool
	bufferSize int64

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	position int64
	stream   *SegmentsStream

	// ReadAt state, see readAtCursor. ReadAt never takes mu.
	contiguous bool // segment numbers are exactly 1..N
	raMu       sync.Mutex
	ra         *readAtCursor // parked between calls, nil while a call holds it
	raClosed   bool

	closed bool
}

// readAtCursor is the SegmentsStream of an earlier ReadAt call, kept open
// while calls stay sequential. The 7z reader reads stored entries through an
// unbuffered io.SectionReader, so each 32 KB read of an HTTP response is one
// ReadAt here. A stream per call keeps about two segments in flight and
// re-reads the current segment from the segment cache on every call.
//
// A cursor's stream checks that every segment continues the bytes before
// it, so it serves only bytes at their file offsets. When a check fails, or
// the stream ends before the end of the file, that call is served the way a
// stream per call serves it.
//
// A cursor's read-ahead budget only grows: it keeps the largest per-call
// budget it has served until it grows to the stream buffer size.
type readAtCursor struct {
	stream *SegmentsStream
	pos    int64 // file offset of the stream's next byte
	seq    int64 // bytes served to calls that continued where the last call ended
	budget int64 // read-ahead budget given to the stream so far
	grown  bool
}

func NewFileStream(
	ctx context.Context,
	pool *Pool,
	file *nzb.File,
	bufferSize int64,
) (*FileStream, error) {
	if bufferSize <= 0 {
		bufferSize = config.Newz.StreamBufferSize
	}

	firstSegment, err := pool.fetchFirstSegment(ctx, file)
	if err != nil {
		return nil, err
	}
	fileSize := firstSegment.FileSize

	fileLog.Trace("file stream - created", "segment_count", file.SegmentCount(), "file_size", fileSize, "buffer_size", bufferSize)

	avgSegmentSize := int64(0)
	if file.SegmentCount() > 0 {
		avgSegmentSize = fileSize / int64(file.SegmentCount())
	}

	segmentSizeRatio := float64(1)
	if totalSegmentBytes := file.Size(); totalSegmentBytes > 0 {
		segmentSizeRatio = float64(fileSize) / float64(totalSegmentBytes)
	}

	ctx, cancel := context.WithCancel(ctx)

	return &FileStream{
		file:             file,
		fileSize:         fileSize,
		avgSegmentSize:   avgSegmentSize,
		segmentSizeRatio: segmentSizeRatio,
		contiguous:       segmentsContiguous(file.Segments),

		pool:       pool,
		bufferSize: bufferSize,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *FileStream) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, errors.New("file stream is closed")
	}

	if s.position >= s.fileSize {
		return 0, io.EOF
	}

	if s.stream == nil {
		stream, err := s.createSegmentsStream(s.position, s.bufferSize, false)
		if err != nil {
			return 0, err
		}
		s.stream = stream
	}

	n, err = s.stream.Read(p)
	s.position += int64(n)
	return n, err
}

func (s *FileStream) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.fileSize {
		return 0, io.EOF
	}
	if err := s.ctx.Err(); err != nil {
		return 0, err
	}

	if !s.contiguous {
		return s.readAtOnce(p, off)
	}

	cursor, reused, err := s.takeReadAtCursor(off, int64(len(p)))
	if err != nil {
		if s.ctx.Err() == nil && readAtFallsBack(err) {
			return s.readAtOnce(p, off)
		}
		return 0, err
	}

	n, err = io.ReadFull(cursor.stream, p)
	cursor.pos += int64(n)
	if reused {
		cursor.seq += int64(n)
	}
	if err == nil {
		s.parkReadAtCursor(cursor)
		return n, nil
	}
	cursor.stream.Close()

	if s.ctx.Err() != nil {
		return n, s.ctx.Err()
	}
	if off+int64(n) >= s.fileSize && !errors.Is(err, errSegmentOffset) {
		// The call ran into the end of the file.
		return n, err
	}
	if readAtFallsBack(err) {
		return s.readAtOnce(p, off)
	}
	return n, err
}

// readAtFallsBack reports whether a ReadAt call whose cursor failed with err
// is served again the way a stream per call serves it: the stream stopped
// matching file offsets, ended early, as it does for an NZB that lacks its
// last parts, or got another reader's cancellation. The pool shares a fetch
// of one article between readers, on the context of the reader that started
// it, so a reader that closes its stream can cancel a fetch that another one
// is waiting for.
func readAtFallsBack(err error) bool {
	return errors.Is(err, errSegmentOffset) || err == io.EOF || err == io.ErrUnexpectedEOF || errors.Is(err, context.Canceled)
}

// readAtOnce serves one ReadAt call from a stream of its own. ReadAt uses it
// when the segment numbers are not exactly 1..N: a stream that runs on past a
// duplicate or missing part no longer matches file offsets, while a stream
// located for each call does.
func (s *FileStream) readAtOnce(p []byte, off int64) (int, error) {
	// Use at least the requested read size as buffer, plus one extra segment for overhead
	bufferSize := int64(len(p)) + s.avgSegmentSize
	stream, err := s.createSegmentsStream(off, bufferSize, false)
	if err == nil {
		defer stream.Close()
		var n int
		n, err = io.ReadFull(stream, p)
		if err == nil {
			return n, nil
		}
		if s.ctx.Err() == nil {
			return n, err
		}
		return n, s.ctx.Err()
	}
	if s.ctx.Err() != nil {
		return 0, s.ctx.Err()
	}
	return 0, err
}

// takeReadAtCursor returns the parked cursor if it stands at off, otherwise a
// new cursor at off, and whether the cursor continues an earlier call. The
// caller owns the cursor until it parks or closes it, so concurrent ReadAt
// calls never share a stream, and no lock is held while a call waits on a
// provider.
func (s *FileStream) takeReadAtCursor(off, size int64) (*readAtCursor, bool, error) {
	s.raMu.Lock()
	if s.raClosed {
		s.raMu.Unlock()
		return nil, false, errors.New("file stream is closed")
	}
	cursor := s.ra
	s.ra = nil
	s.raMu.Unlock()

	// Use at least the requested read size as buffer, plus one extra segment for overhead
	budget := size + s.avgSegmentSize

	if cursor != nil {
		if cursor.pos == off {
			// A larger call gets the budget a stream of its own would get.
			if budget > cursor.budget {
				cursor.stream.growBuffer(budget - cursor.budget)
				cursor.budget = budget
			}
			return cursor, true, nil
		}
		cursor.stream.Close()
	}

	stream, err := s.createSegmentsStream(off, budget, true)
	if err != nil {
		return nil, false, err
	}
	return &readAtCursor{stream: stream, pos: off, budget: budget}, false, nil
}

// parkReadAtCursor keeps the cursor for the next ReadAt call. Once calls that
// continue where the last one ended have read two average segments from a
// cursor, its read-ahead grows to the stream buffer size, as Read's does.
// Header reads, random access and single large reads keep the per-call
// budget.
func (s *FileStream) parkReadAtCursor(cursor *readAtCursor) {
	s.raMu.Lock()
	if s.raClosed || s.ra != nil {
		s.raMu.Unlock()
		cursor.stream.Close()
		return
	}
	// Grow only a cursor that is kept, so a closed one never starts fetches.
	if !cursor.grown && cursor.seq >= 2*s.avgSegmentSize {
		cursor.grown = true
		if s.bufferSize > cursor.budget {
			cursor.stream.growBuffer(s.bufferSize - cursor.budget)
			cursor.budget = s.bufferSize
		}
	}
	s.ra = cursor
	s.raMu.Unlock()
}

// segmentsContiguous reports whether the segment numbers are exactly 1..N.
// NZB parsing sorts segments by number but keeps duplicates and gaps.
func segmentsContiguous(segments []nzb.Segment) bool {
	for i := range segments {
		if segments[i].Number != i+1 {
			return false
		}
	}
	return true
}

func (s *FileStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileLog.Trace("file stream - seek", "offset", offset, "whence", whence)

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

	if newPos != s.position {
		fileLog.Trace("file stream - seek position changed", "old_position", s.position, "new_position", newPos, "whence", whence)
		if s.stream != nil {
			s.stream.Close()
			s.stream = nil
		}
		s.position = newPos
	}

	return s.position, nil
}

func (s *FileStream) Size() int64 {
	return s.fileSize
}

func (s *FileStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Close ReadAt before cancelling, so a call that starts now gets an
	// error rather than a cancelled stream. A call that already holds a
	// cursor closes it itself.
	s.raMu.Lock()
	s.raClosed = true
	cursor := s.ra
	s.ra = nil
	s.raMu.Unlock()

	s.cancel()

	if cursor != nil {
		cursor.stream.Close()
	}

	if s.stream != nil {
		return s.stream.Close()
	}
	return nil
}

// createSegmentsStream starts a stream at startPos. With checkOffsets, the
// stream fails with errSegmentOffset when a segment does not continue the
// bytes before it.
func (s *FileStream) createSegmentsStream(startPos int64, bufferSize int64, checkOffsets bool) (*SegmentsStream, error) {
	fileLog.Trace("create segments stream - start", "position", startPos)

	if startPos == 0 {
		stream := NewSegmentsStream(s.ctx, s.pool, s.file.Segments, s.file.Groups, bufferSize)
		if checkOffsets {
			stream.checkSegmentOffsets(0)
		}
		return stream, nil
	}

	result, err := s.interpolationSearch(startPos)
	if err != nil {
		return nil, fmt.Errorf("failed to find segment for position %d: %w", startPos, err)
	}

	fileLog.Trace("create segments stream - found segment", "segment_idx", result.SegmentIndex, "byte_range", fmt.Sprintf("[%d, %d)", result.ByteRange.Start, result.ByteRange.End))

	stream := NewSegmentsStream(s.ctx, s.pool, s.file.Segments[result.SegmentIndex:], s.file.Groups, bufferSize)
	if checkOffsets {
		stream.checkSegmentOffsets(result.ByteRange.Start)
	}

	skipBytes := startPos - result.ByteRange.Start
	if skipBytes > 0 {
		fileLog.Trace("create segments stream - skipping bytes", "skip_bytes", skipBytes)
		if _, err := io.CopyN(io.Discard, stream, skipBytes); err != nil {
			if err == io.EOF {
				return stream, nil
			}
			stream.Close()
			return nil, fmt.Errorf("failed to skip %d bytes: %w", skipBytes, err)
		}
	}

	return stream, nil
}

func (s *FileStream) getSegmentByteRange(ctx context.Context, index int) (ByteRange, error) {
	segment := &s.file.Segments[index]

	fileLog.Trace("file stream - get segment byte range", "segment_num", segment.Number, "message_id", segment.MessageId)

	data, err := s.pool.fetchSegment(ctx, segment, s.file.Groups)
	if err != nil {
		return ByteRange{}, err
	}

	byteRange := data.ByteRange
	fileLog.Trace("file stream - segment byte range", "segment_num", segment.Number, "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))

	return byteRange, nil
}

func (s *FileStream) estimateSegmentIndex(targetByte int64) int {
	var offset int64
	for i := range s.file.Segments {
		segBytes := s.file.Segments[i].Bytes
		if segBytes <= 0 {
			continue
		}
		estimatedDecodedBytes := int64(float64(segBytes) * s.segmentSizeRatio)
		if targetByte < offset+estimatedDecodedBytes {
			return i
		}
		offset += estimatedDecodedBytes
	}
	return len(s.file.Segments) - 1
}

type searchResult struct {
	SegmentIndex int
	ByteRange    ByteRange
}

func (s *FileStream) interpolationSearch(targetByte int64) (searchResult, error) {
	segmentCount := s.file.SegmentCount()

	if segmentCount == 0 {
		return searchResult{}, fmt.Errorf("no segments to search")
	}

	if targetByte < 0 || targetByte >= s.fileSize {
		return searchResult{}, fmt.Errorf("target byte %d out of bounds [0, %d)", targetByte, s.fileSize)
	}

	indexRange := ByteRange{Start: 0, End: int64(segmentCount)}
	byteRange := ByteRange{Start: 0, End: s.fileSize}

	estimatedIdx := s.estimateSegmentIndex(targetByte)
	fileLog.Trace("search - started", "target_byte", targetByte, "segment_count", segmentCount, "file_size", s.fileSize, "initial_guess", estimatedIdx)
	if estimatedIdx >= 0 && estimatedIdx < segmentCount {
		segmentRange, err := s.getSegmentByteRange(s.ctx, estimatedIdx)
		if err == nil && segmentRange.Contains(targetByte) {
			fileLog.Trace("search - found via initial guess", "segment_idx", estimatedIdx, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))
			return searchResult{SegmentIndex: estimatedIdx, ByteRange: segmentRange}, nil
		}
		// Initial guess was wrong, narrow search bounds if we got a valid range
		if err == nil {
			if targetByte < segmentRange.Start {
				indexRange = ByteRange{Start: 0, End: int64(estimatedIdx)}
				byteRange = ByteRange{Start: 0, End: segmentRange.Start}
			} else {
				indexRange = ByteRange{Start: int64(estimatedIdx + 1), End: int64(segmentCount)}
				byteRange = ByteRange{Start: segmentRange.End, End: s.fileSize}
			}
			fileLog.Trace("search - narrowed bounds", "index_range", fmt.Sprintf("[%d, %d)", indexRange.Start, indexRange.End), "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))
		}
	}

	for {
		select {
		case <-s.ctx.Done():
			return searchResult{}, s.ctx.Err()
		default:
		}

		// Validate search is possible
		if !byteRange.Contains(targetByte) || indexRange.Count() <= 0 {
			return searchResult{}, fmt.Errorf("cannot find byte %d in range [%d, %d)",
				targetByte, byteRange.Start, byteRange.End)
		}

		// Estimate segment based on average bytes per segment
		bytesPerSegment := float64(byteRange.Count()) / float64(indexRange.Count())
		offsetFromStart := float64(targetByte - byteRange.Start)
		guessedOffset := int64(offsetFromStart / bytesPerSegment)
		guessedIndex := int(indexRange.Start + guessedOffset)

		// Clamp to valid range
		if guessedIndex < int(indexRange.Start) {
			guessedIndex = int(indexRange.Start)
		}
		if guessedIndex >= int(indexRange.End) {
			guessedIndex = int(indexRange.End) - 1
		}

		fileLog.Trace("search - probing", "guessed_idx", guessedIndex)

		// Fetch actual byte range of guessed segment
		segmentRange, err := s.getSegmentByteRange(s.ctx, guessedIndex)
		if err != nil {
			return searchResult{}, fmt.Errorf("failed to get byte range for segment %d: %w", guessedIndex, err)
		}

		fileLog.Trace("search - segment range", "segment_idx", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))

		// Validate segment range is within expected bounds
		if !byteRange.ContainsRange(segmentRange) {
			return searchResult{}, fmt.Errorf("corrupt file: segment %d range [%d, %d) outside expected [%d, %d)",
				guessedIndex, segmentRange.Start, segmentRange.End, byteRange.Start, byteRange.End)
		}

		// Check if we found the target
		if segmentRange.Contains(targetByte) {
			fileLog.Trace("search - found", "segment_idx", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))
			return searchResult{SegmentIndex: guessedIndex, ByteRange: segmentRange}, nil
		}

		// Adjust search bounds
		if targetByte < segmentRange.Start {
			// Guessed too high, search lower
			fileLog.Trace("search - adjusting", "direction", "lower", "new_index_range", fmt.Sprintf("[%d, %d)", indexRange.Start, guessedIndex))
			indexRange = ByteRange{Start: indexRange.Start, End: int64(guessedIndex)}
			byteRange = ByteRange{Start: byteRange.Start, End: segmentRange.Start}
		} else {
			// Guessed too low, search higher
			fileLog.Trace("search - adjusting", "direction", "higher", "new_index_range", fmt.Sprintf("[%d, %d)", guessedIndex+1, indexRange.End))
			indexRange = ByteRange{Start: int64(guessedIndex + 1), End: indexRange.End}
			byteRange = ByteRange{Start: segmentRange.End, End: byteRange.End}
		}
	}
}
