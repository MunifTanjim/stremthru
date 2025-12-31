package usenet_pool

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

// FilePart represents one part of a multi-part archive file
type FilePart struct {
	Segments          []string  // Segment message IDs for this archive part
	SegmentByteRange  ByteRange // Total byte range covered by segments
	FilePartByteRange ByteRange // Where the actual file data is within segments
}

// ProcessedFile represents a file ready for streaming
type ProcessedFile struct {
	Name   string
	Size   int64
	Type   FileType
	Groups []string // Newsgroups for fetching

	// For direct files (FileTypeDirect)
	Segments []string

	// For archived files (FileTypeRar, FileType7z)
	FileParts []FilePart
	AesParams *AesParams // Non-nil if encrypted (7z only)
}
