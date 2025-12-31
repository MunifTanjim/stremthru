package usenet_pool

import (
	"context"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

var searchLog = logger.Scoped("nntp/search")

// SearchResult contains the result of an interpolation search
type SearchResult struct {
	Index     int       // Segment index
	ByteRange ByteRange // Byte range this segment covers
}

// GetByteRangeFunc is a function that fetches the byte range for a segment
type GetByteRangeFunc func(ctx context.Context, index int) (ByteRange, error)

// InterpolationSearch finds which segment contains the target byte offset
// This is O(log log N) on average for uniformly distributed data
func InterpolationSearch(
	ctx context.Context,
	targetByte int64,
	segmentCount int,
	fileSize int64,
	getByteRange GetByteRangeFunc,
) (SearchResult, error) {
	if segmentCount == 0 {
		return SearchResult{}, fmt.Errorf("no segments to search")
	}

	if targetByte < 0 || targetByte >= fileSize {
		return SearchResult{}, fmt.Errorf("target byte %d out of bounds [0, %d)", targetByte, fileSize)
	}

	searchLog.Trace("InterpolationSearch started", "target_byte", targetByte, "segment_count", segmentCount, "file_size", fileSize)

	indexRange := ByteRange{Start: 0, End: int64(segmentCount)}
	byteRange := ByteRange{Start: 0, End: fileSize}

	for {
		select {
		case <-ctx.Done():
			return SearchResult{}, ctx.Err()
		default:
		}

		// Validate search is possible
		if !byteRange.Contains(targetByte) || indexRange.Count() <= 0 {
			return SearchResult{}, fmt.Errorf("cannot find byte %d in range [%d, %d)",
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

		searchLog.Trace("InterpolationSearch probing", "guessed_index", guessedIndex)

		// Fetch actual byte range of guessed segment
		segmentRange, err := getByteRange(ctx, guessedIndex)
		if err != nil {
			return SearchResult{}, fmt.Errorf("failed to get byte range for segment %d: %w", guessedIndex, err)
		}

		searchLog.Trace("InterpolationSearch segment range", "index", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))

		// Validate segment range is within expected bounds
		if !byteRange.ContainsRange(segmentRange) {
			return SearchResult{}, fmt.Errorf("corrupt file: segment %d range [%d, %d) outside expected [%d, %d)",
				guessedIndex, segmentRange.Start, segmentRange.End, byteRange.Start, byteRange.End)
		}

		// Check if we found the target
		if segmentRange.Contains(targetByte) {
			searchLog.Trace("InterpolationSearch found", "index", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))
			return SearchResult{Index: guessedIndex, ByteRange: segmentRange}, nil
		}

		// Adjust search bounds
		if targetByte < segmentRange.Start {
			// Guessed too high, search lower
			searchLog.Trace("InterpolationSearch adjusting", "direction", "lower", "new_index_range", fmt.Sprintf("[%d, %d)", indexRange.Start, guessedIndex))
			indexRange = ByteRange{Start: indexRange.Start, End: int64(guessedIndex)}
			byteRange = ByteRange{Start: byteRange.Start, End: segmentRange.Start}
		} else {
			// Guessed too low, search higher
			searchLog.Trace("InterpolationSearch adjusting", "direction", "higher", "new_index_range", fmt.Sprintf("[%d, %d)", guessedIndex+1, indexRange.End))
			indexRange = ByteRange{Start: int64(guessedIndex + 1), End: indexRange.End}
			byteRange = ByteRange{Start: segmentRange.End, End: byteRange.End}
		}
	}
}
