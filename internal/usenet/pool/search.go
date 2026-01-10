package usenet_pool

import (
	"context"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

var searchLog = logger.Scoped("usenet/pool/search")

type SearchResult struct {
	SegmentIndex int
	ByteRange    ByteRange
}

type GetByteRangeFunc func(ctx context.Context, index int) (ByteRange, error)

type InterpolationSearchParams struct {
	TargetByte            int64
	FileSize              int64
	SegmentCount          int
	EstimatedSegmentIndex int
}

func InterpolationSearch(
	ctx context.Context,
	params InterpolationSearchParams,
	getByteRange GetByteRangeFunc,
) (SearchResult, error) {
	if params.SegmentCount == 0 {
		return SearchResult{}, fmt.Errorf("no segments to search")
	}

	if params.TargetByte < 0 || params.TargetByte >= params.FileSize {
		return SearchResult{}, fmt.Errorf("target byte %d out of bounds [0, %d)", params.TargetByte, params.FileSize)
	}

	searchLog.Trace("search - started", "target_byte", params.TargetByte, "segment_count", params.SegmentCount, "file_size", params.FileSize, "initial_guess", params.EstimatedSegmentIndex)

	indexRange := ByteRange{Start: 0, End: int64(params.SegmentCount)}
	byteRange := ByteRange{Start: 0, End: params.FileSize}

	// Try initial guess first if provided
	if params.EstimatedSegmentIndex >= 0 && params.EstimatedSegmentIndex < params.SegmentCount {
		segmentRange, err := getByteRange(ctx, params.EstimatedSegmentIndex)
		if err == nil && segmentRange.Contains(params.TargetByte) {
			searchLog.Trace("search - found via initial guess", "segment_idx", params.EstimatedSegmentIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))
			return SearchResult{SegmentIndex: params.EstimatedSegmentIndex, ByteRange: segmentRange}, nil
		}
		// Initial guess was wrong, narrow search bounds if we got a valid range
		if err == nil {
			if params.TargetByte < segmentRange.Start {
				indexRange = ByteRange{Start: 0, End: int64(params.EstimatedSegmentIndex)}
				byteRange = ByteRange{Start: 0, End: segmentRange.Start}
			} else {
				indexRange = ByteRange{Start: int64(params.EstimatedSegmentIndex + 1), End: int64(params.SegmentCount)}
				byteRange = ByteRange{Start: segmentRange.End, End: params.FileSize}
			}
			searchLog.Trace("search - narrowed bounds", "index_range", fmt.Sprintf("[%d, %d)", indexRange.Start, indexRange.End), "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))
		}
	}

	for {
		select {
		case <-ctx.Done():
			return SearchResult{}, ctx.Err()
		default:
		}

		// Validate search is possible
		if !byteRange.Contains(params.TargetByte) || indexRange.Count() <= 0 {
			return SearchResult{}, fmt.Errorf("cannot find byte %d in range [%d, %d)",
				params.TargetByte, byteRange.Start, byteRange.End)
		}

		// Estimate segment based on average bytes per segment
		bytesPerSegment := float64(byteRange.Count()) / float64(indexRange.Count())
		offsetFromStart := float64(params.TargetByte - byteRange.Start)
		guessedOffset := int64(offsetFromStart / bytesPerSegment)
		guessedIndex := int(indexRange.Start + guessedOffset)

		// Clamp to valid range
		if guessedIndex < int(indexRange.Start) {
			guessedIndex = int(indexRange.Start)
		}
		if guessedIndex >= int(indexRange.End) {
			guessedIndex = int(indexRange.End) - 1
		}

		searchLog.Trace("search - probing", "guessed_idx", guessedIndex)

		// Fetch actual byte range of guessed segment
		segmentRange, err := getByteRange(ctx, guessedIndex)
		if err != nil {
			return SearchResult{}, fmt.Errorf("failed to get byte range for segment %d: %w", guessedIndex, err)
		}

		searchLog.Trace("search - segment range", "segment_idx", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))

		// Validate segment range is within expected bounds
		if !byteRange.ContainsRange(segmentRange) {
			return SearchResult{}, fmt.Errorf("corrupt file: segment %d range [%d, %d) outside expected [%d, %d)",
				guessedIndex, segmentRange.Start, segmentRange.End, byteRange.Start, byteRange.End)
		}

		// Check if we found the target
		if segmentRange.Contains(params.TargetByte) {
			searchLog.Trace("search - found", "segment_idx", guessedIndex, "byte_range", fmt.Sprintf("[%d, %d)", segmentRange.Start, segmentRange.End))
			return SearchResult{SegmentIndex: guessedIndex, ByteRange: segmentRange}, nil
		}

		// Adjust search bounds
		if params.TargetByte < segmentRange.Start {
			// Guessed too high, search lower
			searchLog.Trace("search - adjusting", "direction", "lower", "new_index_range", fmt.Sprintf("[%d, %d)", indexRange.Start, guessedIndex))
			indexRange = ByteRange{Start: indexRange.Start, End: int64(guessedIndex)}
			byteRange = ByteRange{Start: byteRange.Start, End: segmentRange.Start}
		} else {
			// Guessed too low, search higher
			searchLog.Trace("search - adjusting", "direction", "higher", "new_index_range", fmt.Sprintf("[%d, %d)", guessedIndex+1, indexRange.End))
			indexRange = ByteRange{Start: int64(guessedIndex + 1), End: indexRange.End}
			byteRange = ByteRange{Start: segmentRange.End, End: byteRange.End}
		}
	}
}
