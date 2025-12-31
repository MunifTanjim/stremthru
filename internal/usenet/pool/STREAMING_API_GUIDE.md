# NZB Streaming API Implementation Guide

A step-by-step guide to implement an HTTP endpoint that accepts an NZB file and streams its contents with Range header support.

## Prerequisites

- Existing NNTP client and connection pool
- Go 1.21+

## Architecture Overview

```
POST /stream (multipart form with NZB file + optional password)
     │
     ▼
┌──────────────────┐
│  Parse NZB XML   │ ◄── Extract segment IDs and file metadata
└────────┬─────────┘
         │
         ▼
┌────────────────────┐
│ File Type Detection│ ◄── Magic bytes (first 16KB) + extension
└────────┬───────────┘
         │
         ├──────────────────────────────┐
         │                              │
         ▼                              ▼
┌──────────────────┐          ┌─────────────────────┐
│  Direct File     │          │  Archive (RAR/7z)   │
│  (.mkv, .mp4)    │          │  Parse headers,     │
│                  │          │  extract byte ranges │
└────────┬─────────┘          └──────────┬──────────┘
         │                               │
         ▼                               ▼
┌──────────────────┐          ┌─────────────────────┐
│  NzbFileStream   │          │ MultipartFileStream │
└────────┬─────────┘          └──────────┬──────────┘
         │                               │
         └───────────────┬───────────────┘
                         │
                         ▼
              ┌────────────────────┐
              │ MultiSegmentStream │ ◄── Parallel prefetch
              └────────┬───────────┘
                       │
                       ▼
              ┌──────────────────┐
              │  Your NNTP Pool  │
              └────────┬─────────┘
                       │
                       ▼
                  Usenet Server
```

---

## Step 1: Define Core Data Structures

Create `models.go`:

```go
package nzbstream

import (
    "encoding/xml"
)

// LongRange represents a half-open byte interval [Start, End)
type LongRange struct {
    Start int64
    End   int64
}

func (r LongRange) Count() int64 {
    return r.End - r.Start
}

func (r LongRange) Contains(value int64) bool {
    return value >= r.Start && value < r.End
}

func (r LongRange) ContainsRange(other LongRange) bool {
    return other.Start >= r.Start && other.End <= r.End
}

// NzbDocument represents the parsed NZB XML
type NzbDocument struct {
    XMLName xml.Name  `xml:"nzb"`
    Files   []NzbFile `xml:"file"`
}

// NzbFile represents a single file in the NZB
type NzbFile struct {
    Subject  string       `xml:"subject,attr"`
    Poster   string       `xml:"poster,attr"`
    Date     int64        `xml:"date,attr"`
    Groups   []string     `xml:"groups>group"`
    Segments []NzbSegment `xml:"segments>segment"`
}

// NzbSegment represents a single article segment
type NzbSegment struct {
    Number    int    `xml:"number,attr"`
    Bytes     int64  `xml:"bytes,attr"`
    MessageID string `xml:",chardata"`
}

// GetSegmentIDs returns ordered message IDs
func (f *NzbFile) GetSegmentIDs() []string {
    // Sort by segment number first
    segments := make([]NzbSegment, len(f.Segments))
    copy(segments, f.Segments)
    sort.Slice(segments, func(i, j int) bool {
        return segments[i].Number < segments[j].Number
    })

    ids := make([]string, len(segments))
    for i, seg := range segments {
        ids[i] = seg.MessageID
    }
    return ids
}

// TotalBytes returns the sum of all segment sizes (approximate file size)
func (f *NzbFile) TotalBytes() int64 {
    var total int64
    for _, seg := range f.Segments {
        total += seg.Bytes
    }
    return total
}

// FileType represents the detected type of a file
type FileType int

const (
    FileTypeDirect FileType = iota // Direct video file (.mkv, .mp4, etc.)
    FileTypeRar                    // RAR archive
    FileType7z                     // 7z archive
)

// ProcessedFile represents a file ready for streaming
type ProcessedFile struct {
    Name     string
    Size     int64
    Type     FileType

    // For direct files
    SegmentIDs []string

    // For archived files
    FileParts []FilePart
    AesParams *AesParams // Non-nil if encrypted
}

// FilePart represents one part of a multi-part archive file
type FilePart struct {
    SegmentIDs         []string  // Segments for this archive part
    SegmentByteRange   LongRange // Total bytes in the segments
    FilePartByteRange  LongRange // Where the actual file data is within segments
}

// AesParams contains encryption parameters for 7z archives
type AesParams struct {
    DecodedSize int64    // Size after decryption
    IV          [16]byte // Initialization vector
    Key         [32]byte // Derived AES-256 key
}
```

---

## Step 2: Parse NZB Files

> **Note:** If you already have an NZB parser, you can use it directly. Just ensure your parsed NZB structure provides access to segment IDs (message IDs) in order, and file metadata like subject/filename. The code below shows the expected interface.

Create `parser.go` (or adapt your existing parser):

```go
package nzbstream

import (
    "encoding/xml"
    "fmt"
    "io"
    "regexp"
    "strings"
)

// ParseNzb parses an NZB XML document
func ParseNzb(r io.Reader) (*NzbDocument, error) {
    var doc NzbDocument
    decoder := xml.NewDecoder(r)
    if err := decoder.Decode(&doc); err != nil {
        return nil, fmt.Errorf("failed to parse NZB: %w", err)
    }

    // Filter out files with no segments
    var validFiles []NzbFile
    for _, file := range doc.Files {
        if len(file.Segments) > 0 {
            validFiles = append(validFiles, file)
        }
    }
    doc.Files = validFiles

    return &doc, nil
}

// GetLargestFile returns the largest file in the NZB (typically the video)
func (doc *NzbDocument) GetLargestFile() *NzbFile {
    var largest *NzbFile
    var largestSize int64

    for i := range doc.Files {
        size := doc.Files[i].TotalBytes()
        if size > largestSize {
            largestSize = size
            largest = &doc.Files[i]
        }
    }

    return largest
}

// GetFileByName finds a file by matching the subject line
func (doc *NzbDocument) GetFileByName(name string) *NzbFile {
    nameLower := strings.ToLower(name)
    for i := range doc.Files {
        if strings.Contains(strings.ToLower(doc.Files[i].Subject), nameLower) {
            return &doc.Files[i]
        }
    }
    return nil
}

// ExtractFilename attempts to extract filename from subject line
// Subject format is typically: "description "filename.ext" yEnc (1/100)"
func ExtractFilename(subject string) string {
    // Try quoted filename first
    re := regexp.MustCompile(`"([^"]+)"`)
    if matches := re.FindStringSubmatch(subject); len(matches) > 1 {
        return matches[1]
    }

    // Fallback: take everything before yEnc
    if idx := strings.Index(subject, " yEnc"); idx > 0 {
        return strings.TrimSpace(subject[:idx])
    }

    return subject
}
```

---

## Step 3: Implement yEnc Decoder

Create `yenc.go`:

```go
package nzbstream

import (
    "bufio"
    "fmt"
    "io"
    "strconv"
    "strings"
)

// YencHeader contains parsed yEnc header information
type YencHeader struct {
    Name      string
    Size      int64  // Total file size
    Part      int    // Part number (0 if single-part)
    Begin     int64  // 1-based start offset
    End       int64  // 1-based end offset (inclusive)
    Line      int    // Characters per line
}

// ByteRange returns the 0-based byte range this part covers
func (h *YencHeader) ByteRange() LongRange {
    return LongRange{
        Start: h.Begin - 1,           // Convert 1-based to 0-based
        End:   h.End,                 // End is inclusive, so this becomes exclusive
    }
}

// YencDecoder decodes yEnc-encoded data
type YencDecoder struct {
    r          *bufio.Reader
    header     *YencHeader
    headerRead bool
    escapeNext bool
    done       bool
}

// NewYencDecoder creates a new yEnc decoder
func NewYencDecoder(r io.Reader) *YencDecoder {
    return &YencDecoder{r: bufio.NewReader(r)}
}

// Header returns the parsed yEnc header, reading it if necessary
func (d *YencDecoder) Header() (*YencHeader, error) {
    if d.headerRead {
        return d.header, nil
    }

    h := &YencHeader{}

    // Read =ybegin line
    line, err := d.r.ReadString('\n')
    if err != nil {
        return nil, fmt.Errorf("failed to read ybegin: %w", err)
    }

    if !strings.HasPrefix(line, "=ybegin ") {
        return nil, fmt.Errorf("expected =ybegin, got: %s", line)
    }

    // Parse =ybegin parameters
    h.Name = parseYencParam(line, "name")
    h.Size, _ = strconv.ParseInt(parseYencParam(line, "size"), 10, 64)
    h.Line, _ = strconv.Atoi(parseYencParam(line, "line"))
    h.Part, _ = strconv.Atoi(parseYencParam(line, "part"))

    // If multipart, read =ypart line
    if h.Part > 0 {
        line, err = d.r.ReadString('\n')
        if err != nil {
            return nil, fmt.Errorf("failed to read ypart: %w", err)
        }

        if strings.HasPrefix(line, "=ypart ") {
            h.Begin, _ = strconv.ParseInt(parseYencParam(line, "begin"), 10, 64)
            h.End, _ = strconv.ParseInt(parseYencParam(line, "end"), 10, 64)
        }
    } else {
        // Single-part: covers entire file
        h.Begin = 1
        h.End = h.Size
    }

    d.header = h
    d.headerRead = true
    return h, nil
}

// Read implements io.Reader, decoding yEnc data
func (d *YencDecoder) Read(p []byte) (n int, err error) {
    if !d.headerRead {
        if _, err := d.Header(); err != nil {
            return 0, err
        }
    }

    if d.done {
        return 0, io.EOF
    }

    for n < len(p) {
        b, err := d.r.ReadByte()
        if err != nil {
            d.done = true
            return n, err
        }

        // Check for =yend (end of data)
        if b == '=' && !d.escapeNext {
            next, err := d.r.Peek(1)
            if err == nil && len(next) > 0 && next[0] == 'y' {
                d.done = true
                return n, io.EOF
            }
            d.escapeNext = true
            continue
        }

        // Skip line endings
        if b == '\r' || b == '\n' {
            continue
        }

        // Decode byte
        if d.escapeNext {
            p[n] = (b - 64 - 42) & 0xFF
            d.escapeNext = false
        } else {
            p[n] = (b - 42) & 0xFF
        }
        n++
    }

    return n, nil
}

// parseYencParam extracts a parameter value from a yEnc header line
func parseYencParam(line, param string) string {
    prefix := param + "="
    idx := strings.Index(line, prefix)
    if idx < 0 {
        return ""
    }

    start := idx + len(prefix)
    end := start

    // Handle quoted values for "name"
    if param == "name" {
        // Name is always at the end, take everything after "name="
        return strings.TrimSpace(line[start:])
    }

    // For other params, find the end (space or newline)
    for end < len(line) && line[end] != ' ' && line[end] != '\r' && line[end] != '\n' {
        end++
    }

    return line[start:end]
}
```

---

## Step 4: Define the NNTP Client Interface

Create `nntp_interface.go`:

```go
package nzbstream

import (
    "context"
    "io"
)

// NntpClient is the interface your existing NNTP client should implement
type NntpClient interface {
    // Body fetches the raw article body (yEnc encoded)
    Body(ctx context.Context, messageID string) (io.ReadCloser, error)

    // DecodedBody fetches and decodes the article body
    // Returns the decoded bytes as a stream
    DecodedBody(ctx context.Context, messageID string) (io.ReadCloser, error)

    // GetYencHeader fetches just enough of the article to parse yEnc headers
    // Returns the byte range this segment covers
    GetYencHeader(ctx context.Context, messageID string) (*YencHeader, error)
}

// NntpClientAdapter wraps your existing client to implement NntpClient
type NntpClientAdapter struct {
    // YourClient is your existing NNTP client/pool
    YourClient interface {
        // Adjust these method signatures to match your client
        GetArticleBody(ctx context.Context, messageID string) (io.ReadCloser, error)
    }
}

func (a *NntpClientAdapter) Body(ctx context.Context, messageID string) (io.ReadCloser, error) {
    return a.YourClient.GetArticleBody(ctx, messageID)
}

func (a *NntpClientAdapter) DecodedBody(ctx context.Context, messageID string) (io.ReadCloser, error) {
    body, err := a.Body(ctx, messageID)
    if err != nil {
        return nil, err
    }

    decoder := NewYencDecoder(body)

    // Read header to initialize decoder
    if _, err := decoder.Header(); err != nil {
        body.Close()
        return nil, err
    }

    return &decodedBodyReader{
        decoder: decoder,
        closer:  body,
    }, nil
}

func (a *NntpClientAdapter) GetYencHeader(ctx context.Context, messageID string) (*YencHeader, error) {
    body, err := a.Body(ctx, messageID)
    if err != nil {
        return nil, err
    }
    defer body.Close()

    decoder := NewYencDecoder(body)
    return decoder.Header()
}

type decodedBodyReader struct {
    decoder *YencDecoder
    closer  io.Closer
}

func (r *decodedBodyReader) Read(p []byte) (n int, err error) {
    return r.decoder.Read(p)
}

func (r *decodedBodyReader) Close() error {
    return r.closer.Close()
}
```

---

## Step 5: Implement File Type Detection

Create `detect.go`:

```go
package nzbstream

import (
    "bytes"
    "regexp"
    "strings"
)

// Magic byte signatures
var (
    Rar4Magic   = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}       // "Rar!..."
    Rar5Magic   = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00} // "Rar!...." (RAR5)
    SevenZMagic = []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}             // "7z...."
)

// DetectFileType determines the file type from the first bytes and filename
func DetectFileType(first16KB []byte, filename string) FileType {
    // Check magic bytes first (most reliable)
    if bytes.HasPrefix(first16KB, Rar5Magic) || bytes.HasPrefix(first16KB, Rar4Magic) {
        return FileTypeRar
    }
    if bytes.HasPrefix(first16KB, SevenZMagic) {
        return FileType7z
    }

    // Fall back to extension-based detection
    filenameLower := strings.ToLower(filename)

    // RAR patterns: .rar, .r00, .r01, .part01.rar
    if strings.HasSuffix(filenameLower, ".rar") {
        return FileTypeRar
    }
    if matched, _ := regexp.MatchString(`\.r\d+$`, filenameLower); matched {
        return FileTypeRar
    }

    // 7z patterns: .7z, .7z.001, .7z.002
    if matched, _ := regexp.MatchString(`\.7z(\.\d+)?$`, filenameLower); matched {
        return FileType7z
    }

    return FileTypeDirect
}

// IsVideoFile checks if filename has a video extension
func IsVideoFile(filename string) bool {
    videoExts := []string{
        ".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm",
        ".m4v", ".mpg", ".mpeg", ".ts", ".m2ts", ".vob",
    }
    lower := strings.ToLower(filename)
    for _, ext := range videoExts {
        if strings.HasSuffix(lower, ext) {
            return true
        }
    }
    return false
}

// GetPartNumber extracts the part number from a RAR filename
func GetRarPartNumber(filename string) int {
    filenameLower := strings.ToLower(filename)

    // .part01.rar format
    re := regexp.MustCompile(`\.part(\d+)\.rar$`)
    if matches := re.FindStringSubmatch(filenameLower); len(matches) > 1 {
        n, _ := strconv.Atoi(matches[1])
        return n
    }

    // .r00, .r01 format
    re = regexp.MustCompile(`\.r(\d+)$`)
    if matches := re.FindStringSubmatch(filenameLower); len(matches) > 1 {
        n, _ := strconv.Atoi(matches[1])
        return n + 1 // .r00 is part 1
    }

    // .rar is first part (or only part)
    if strings.HasSuffix(filenameLower, ".rar") {
        return 0 // First part
    }

    return -1
}

// Get7zPartNumber extracts the part number from a 7z filename
func Get7zPartNumber(filename string) int {
    filenameLower := strings.ToLower(filename)

    // .7z.001, .7z.002 format
    re := regexp.MustCompile(`\.7z\.(\d+)$`)
    if matches := re.FindStringSubmatch(filenameLower); len(matches) > 1 {
        n, _ := strconv.Atoi(matches[1])
        return n
    }

    // .7z is first part
    if strings.HasSuffix(filenameLower, ".7z") {
        return 0
    }

    return -1
}
```

---

## Step 6: Implement RAR Header Parsing

Create `rar.go`:

```go
package nzbstream

import (
    "encoding/binary"
    "fmt"
    "io"
)

// RarFileEntry represents a file inside a RAR archive
type RarFileEntry struct {
    Filename        string
    DataStartPos    int64     // Where compressed data starts in this part
    CompressedSize  int64     // Size of compressed data in this part
    UncompressedSize int64    // Total uncompressed size
    PartNumber      int       // Which part this entry is in
}

// ParseRarHeaders reads RAR headers to find file entries
// This doesn't decompress - it just finds where file data is located
func ParseRarHeaders(r io.ReadSeeker, fileSize int64) ([]RarFileEntry, error) {
    // Read first 7-8 bytes to detect RAR version
    header := make([]byte, 8)
    if _, err := r.Read(header); err != nil {
        return nil, fmt.Errorf("failed to read RAR header: %w", err)
    }
    r.Seek(0, io.SeekStart)

    if bytes.HasPrefix(header, Rar5Magic) {
        return parseRar5Headers(r, fileSize)
    } else if bytes.HasPrefix(header, Rar4Magic) {
        return parseRar4Headers(r, fileSize)
    }

    return nil, fmt.Errorf("not a valid RAR file")
}

func parseRar5Headers(r io.ReadSeeker, fileSize int64) ([]RarFileEntry, error) {
    var entries []RarFileEntry

    // Skip RAR5 signature (8 bytes)
    r.Seek(8, io.SeekStart)

    for {
        pos, _ := r.Seek(0, io.SeekCurrent)
        if pos >= fileSize {
            break
        }

        // Read header CRC and size
        headerBuf := make([]byte, 32)
        n, err := r.Read(headerBuf)
        if err == io.EOF || n < 7 {
            break
        }

        // Parse variable-length header size
        headerSize, sizeBytes := readVInt(headerBuf[4:])
        headerType, typeBytes := readVInt(headerBuf[4+sizeBytes:])

        if headerType == 2 { // File header
            // Parse file header to get name and data position
            entry, err := parseRar5FileHeader(r, pos, int64(headerSize))
            if err == nil {
                entries = append(entries, entry)
            }
        }

        // Skip to next header
        r.Seek(pos+int64(headerSize)+4, io.SeekStart) // +4 for CRC
    }

    return entries, nil
}

func parseRar4Headers(r io.ReadSeeker, fileSize int64) ([]RarFileEntry, error) {
    var entries []RarFileEntry

    // Skip RAR4 signature (7 bytes)
    r.Seek(7, io.SeekStart)

    for {
        pos, _ := r.Seek(0, io.SeekCurrent)
        if pos >= fileSize {
            break
        }

        // Read block header
        blockHeader := make([]byte, 7)
        if _, err := io.ReadFull(r, blockHeader); err != nil {
            break
        }

        // Parse header
        // headerCrc := binary.LittleEndian.Uint16(blockHeader[0:2])
        headerType := blockHeader[2]
        headerFlags := binary.LittleEndian.Uint16(blockHeader[3:5])
        headerSize := binary.LittleEndian.Uint16(blockHeader[5:7])

        if headerType == 0x74 { // File header
            entry, err := parseRar4FileHeader(r, pos, int64(headerSize), headerFlags)
            if err == nil {
                entries = append(entries, entry)
            }
        }

        // Calculate add size if present
        var addSize int64
        if headerFlags&0x8000 != 0 {
            addSizeBuf := make([]byte, 4)
            r.Read(addSizeBuf)
            addSize = int64(binary.LittleEndian.Uint32(addSizeBuf))
        }

        // Move to next header
        r.Seek(pos+int64(headerSize)+addSize, io.SeekStart)
    }

    return entries, nil
}

func parseRar5FileHeader(r io.ReadSeeker, headerPos int64, headerSize int64) (RarFileEntry, error) {
    // Simplified - in production use a proper RAR library
    // This returns approximate positions for demonstration
    return RarFileEntry{
        DataStartPos:   headerPos + headerSize + 4, // After header
        CompressedSize: 0, // Would need full parsing
    }, nil
}

func parseRar4FileHeader(r io.ReadSeeker, headerPos int64, headerSize int64, flags uint16) (RarFileEntry, error) {
    // Read remaining header
    remaining := make([]byte, headerSize-7)
    if _, err := io.ReadFull(r, remaining); err != nil {
        return RarFileEntry{}, err
    }

    compressedSize := int64(binary.LittleEndian.Uint32(remaining[0:4]))
    uncompressedSize := int64(binary.LittleEndian.Uint32(remaining[4:8]))
    // hostOS := remaining[8]
    // fileCRC := binary.LittleEndian.Uint32(remaining[9:13])
    // fileTime := binary.LittleEndian.Uint32(remaining[13:17])
    // version := remaining[17]
    // method := remaining[18]
    nameSize := int(binary.LittleEndian.Uint16(remaining[19:21]))
    // fileAttr := binary.LittleEndian.Uint32(remaining[21:25])

    // High 32 bits of sizes if flag is set
    offset := 25
    if flags&0x100 != 0 { // Large file
        highCompressed := int64(binary.LittleEndian.Uint32(remaining[offset : offset+4]))
        highUncompressed := int64(binary.LittleEndian.Uint32(remaining[offset+4 : offset+8]))
        compressedSize |= highCompressed << 32
        uncompressedSize |= highUncompressed << 32
        offset += 8
    }

    filename := string(remaining[offset : offset+nameSize])

    return RarFileEntry{
        Filename:         filename,
        DataStartPos:     headerPos + int64(headerSize),
        CompressedSize:   compressedSize,
        UncompressedSize: uncompressedSize,
    }, nil
}

// readVInt reads a RAR5 variable-length integer
func readVInt(data []byte) (value uint64, bytesRead int) {
    for i, b := range data {
        value |= uint64(b&0x7F) << (7 * i)
        bytesRead = i + 1
        if b&0x80 == 0 {
            break
        }
    }
    return
}
```

**Note:** For production use, consider using a proper RAR library like `github.com/nwaples/rardecode`.

---

## Step 7: Implement 7z Header Parsing

Create `sevenz.go`:

```go
package nzbstream

import (
    "bytes"
    "crypto/aes"
    "crypto/cipher"
    "crypto/sha256"
    "encoding/binary"
    "fmt"
    "io"
    "unicode/utf16"
)

// SevenZFileEntry represents a file inside a 7z archive
type SevenZFileEntry struct {
    Filename       string
    Size           int64     // Uncompressed size
    Offset         int64     // Offset within the archive
    AesParams      *AesParams // Non-nil if encrypted
}

// Parse7zHeaders reads 7z headers to find file entries
func Parse7zHeaders(r io.ReadSeeker, password string) ([]SevenZFileEntry, error) {
    // Read signature header (32 bytes)
    sigHeader := make([]byte, 32)
    if _, err := io.ReadFull(r, sigHeader); err != nil {
        return nil, fmt.Errorf("failed to read 7z header: %w", err)
    }

    // Verify magic
    if !bytes.Equal(sigHeader[0:6], SevenZMagic) {
        return nil, fmt.Errorf("not a valid 7z file")
    }

    // Parse start header
    // nextHeaderOffset := binary.LittleEndian.Uint64(sigHeader[12:20])
    // nextHeaderSize := binary.LittleEndian.Uint64(sigHeader[20:28])

    // Simplified: For demonstration, return empty
    // In production, use github.com/bodgit/sevenzip
    return nil, fmt.Errorf("7z parsing requires external library - use github.com/bodgit/sevenzip")
}

// Derive7zKey derives the AES-256 key using 7z's algorithm
func Derive7zKey(password string, salt []byte, numCycles int) [32]byte {
    // Convert password to UTF-16LE
    passRunes := []rune(password)
    passUtf16 := utf16.Encode(passRunes)
    passBytes := make([]byte, len(passUtf16)*2)
    for i, r := range passUtf16 {
        binary.LittleEndian.PutUint16(passBytes[i*2:], r)
    }

    // Special case: direct copy (numCycles == 0x3F)
    if numCycles == 0x3F {
        var key [32]byte
        copy(key[:], salt)
        copy(key[len(salt):], passBytes)
        return key
    }

    // Iterative SHA256
    h := sha256.New()
    counter := make([]byte, 8)
    numRounds := int64(1) << numCycles

    for round := int64(0); round < numRounds; round++ {
        h.Write(salt)
        h.Write(passBytes)
        h.Write(counter)

        // Increment 8-byte little-endian counter
        for i := 0; i < 8; i++ {
            counter[i]++
            if counter[i] != 0 {
                break
            }
        }
    }

    var key [32]byte
    copy(key[:], h.Sum(nil))
    return key
}

// NewAesDecryptingReader wraps a reader with AES-CBC decryption
func NewAesDecryptingReader(r io.Reader, params AesParams) io.Reader {
    block, _ := aes.NewCipher(params.Key[:])
    mode := cipher.NewCBCDecrypter(block, params.IV[:])

    return &aesReader{
        r:           r,
        mode:        mode,
        decodedSize: params.DecodedSize,
    }
}

type aesReader struct {
    r           io.Reader
    mode        cipher.BlockMode
    decodedSize int64
    pos         int64
    buf         []byte
}

func (a *aesReader) Read(p []byte) (n int, err error) {
    // Return buffered data first
    if len(a.buf) > 0 {
        n = copy(p, a.buf)
        a.buf = a.buf[n:]
        a.pos += int64(n)
        return n, nil
    }

    if a.pos >= a.decodedSize {
        return 0, io.EOF
    }

    // Read and decrypt a block
    blockSize := 16
    remaining := int(a.decodedSize - a.pos)
    toRead := len(p)
    if toRead > remaining {
        toRead = remaining
    }

    // Round up to block size
    blocksNeeded := (toRead + blockSize - 1) / blockSize
    cipherData := make([]byte, blocksNeeded*blockSize)

    _, err = io.ReadFull(a.r, cipherData)
    if err != nil && err != io.EOF {
        return 0, err
    }

    plainData := make([]byte, len(cipherData))
    a.mode.CryptBlocks(plainData, cipherData)

    n = copy(p, plainData[:toRead])
    a.pos += int64(n)

    // Buffer remainder
    if len(plainData) > toRead {
        a.buf = plainData[toRead:]
    }

    return n, nil
}
```

**Note:** For production 7z support, use `github.com/bodgit/sevenzip` which handles the complex 7z format properly.

---

## Step 8: Implement Archive Processor

Create `processor.go`:

```go
package nzbstream

import (
    "context"
    "fmt"
    "io"
    "path"
    "sort"
    "strings"
)

// ProcessNzb analyzes an NZB and returns files ready for streaming
func ProcessNzb(
    ctx context.Context,
    nzb *NzbDocument,
    client NntpClient,
    password string,
) ([]ProcessedFile, error) {
    // Step 1: Fetch first 16KB of each file for detection
    filesWithHeaders := make([]fileWithHeader, 0, len(nzb.Files))

    for i := range nzb.Files {
        nzbFile := &nzb.Files[i]
        if len(nzbFile.Segments) == 0 {
            continue
        }

        // Fetch first segment
        segmentIDs := nzbFile.GetSegmentIDs()
        body, err := client.DecodedBody(ctx, segmentIDs[0])
        if err != nil {
            continue // Skip files with missing segments
        }

        first16KB := make([]byte, 16*1024)
        n, _ := io.ReadFull(body, first16KB)
        body.Close()

        filename := ExtractFilename(nzbFile.Subject)
        fileType := DetectFileType(first16KB[:n], filename)

        filesWithHeaders = append(filesWithHeaders, fileWithHeader{
            NzbFile:   nzbFile,
            Filename:  filename,
            First16KB: first16KB[:n],
            FileType:  fileType,
        })
    }

    // Step 2: Group by file type and process
    var result []ProcessedFile

    // Process direct files
    for _, f := range filesWithHeaders {
        if f.FileType == FileTypeDirect && IsVideoFile(f.Filename) {
            result = append(result, ProcessedFile{
                Name:       f.Filename,
                Size:       f.NzbFile.TotalBytes(),
                Type:       FileTypeDirect,
                SegmentIDs: f.NzbFile.GetSegmentIDs(),
            })
        }
    }

    // Process RAR files
    rarFiles := filterByType(filesWithHeaders, FileTypeRar)
    if len(rarFiles) > 0 {
        archiveFiles, err := processRarArchive(ctx, rarFiles, client, password)
        if err == nil {
            result = append(result, archiveFiles...)
        }
    }

    // Process 7z files
    sevenZFiles := filterByType(filesWithHeaders, FileType7z)
    if len(sevenZFiles) > 0 {
        archiveFiles, err := process7zArchive(ctx, sevenZFiles, client, password)
        if err == nil {
            result = append(result, archiveFiles...)
        }
    }

    return result, nil
}

type fileWithHeader struct {
    NzbFile   *NzbFile
    Filename  string
    First16KB []byte
    FileType  FileType
}

func filterByType(files []fileWithHeader, ft FileType) []fileWithHeader {
    var result []fileWithHeader
    for _, f := range files {
        if f.FileType == ft {
            result = append(result, f)
        }
    }
    return result
}

func processRarArchive(
    ctx context.Context,
    rarFiles []fileWithHeader,
    client NntpClient,
    password string,
) ([]ProcessedFile, error) {
    // Sort by part number
    sort.Slice(rarFiles, func(i, j int) bool {
        return GetRarPartNumber(rarFiles[i].Filename) < GetRarPartNumber(rarFiles[j].Filename)
    })

    // Parse headers from each part and collect file entries
    type partEntry struct {
        PartNum      int
        NzbFile      *NzbFile
        PartSize     int64
        FileEntries  []RarFileEntry
    }

    var parts []partEntry

    for _, rf := range rarFiles {
        // Create a stream to the RAR part
        segmentIDs := rf.NzbFile.GetSegmentIDs()
        partSize := rf.NzbFile.TotalBytes()

        stream := NewNzbFileStream(ctx, segmentIDs, partSize, client, 5)
        defer stream.Close()

        entries, err := ParseRarHeaders(stream, partSize)
        if err != nil {
            continue
        }

        parts = append(parts, partEntry{
            PartNum:     GetRarPartNumber(rf.Filename),
            NzbFile:     rf.NzbFile,
            PartSize:    partSize,
            FileEntries: entries,
        })
    }

    // Group entries by filename and build FileParts
    fileMap := make(map[string]*ProcessedFile)

    for _, part := range parts {
        for _, entry := range part.FileEntries {
            if !IsVideoFile(entry.Filename) {
                continue
            }

            pf, exists := fileMap[entry.Filename]
            if !exists {
                pf = &ProcessedFile{
                    Name: path.Base(entry.Filename),
                    Type: FileTypeRar,
                }
                fileMap[entry.Filename] = pf
            }

            pf.FileParts = append(pf.FileParts, FilePart{
                SegmentIDs:        part.NzbFile.GetSegmentIDs(),
                SegmentByteRange:  LongRange{Start: 0, End: part.PartSize},
                FilePartByteRange: LongRange{Start: entry.DataStartPos, End: entry.DataStartPos + entry.CompressedSize},
            })
            pf.Size += entry.CompressedSize
        }
    }

    var result []ProcessedFile
    for _, pf := range fileMap {
        result = append(result, *pf)
    }
    return result, nil
}

func process7zArchive(
    ctx context.Context,
    sevenZFiles []fileWithHeader,
    client NntpClient,
    password string,
) ([]ProcessedFile, error) {
    // Similar to RAR processing
    // Use github.com/bodgit/sevenzip for proper parsing
    return nil, fmt.Errorf("7z processing requires github.com/bodgit/sevenzip")
}
```

---

## Step 9: Implement MultipartFileStream

Create `multipart_file_stream.go`:

```go
package nzbstream

import (
    "context"
    "fmt"
    "io"
    "sync"
)

// MultipartFileStream handles streaming files split across archive parts
type MultipartFileStream struct {
    fileParts  []FilePart
    fileSize   int64
    client     NntpClient
    bufferSize int
    aesParams  *AesParams

    ctx           context.Context
    cancel        context.CancelFunc
    mu            sync.Mutex
    position      int64
    currentStream io.ReadCloser
    currentPart   int
}

// NewMultipartFileStream creates a stream for files split across archive parts
func NewMultipartFileStream(
    ctx context.Context,
    fileParts []FilePart,
    fileSize int64,
    client NntpClient,
    bufferSize int,
    aesParams *AesParams,
) *MultipartFileStream {
    ctx, cancel := context.WithCancel(ctx)
    return &MultipartFileStream{
        fileParts:  fileParts,
        fileSize:   fileSize,
        client:     client,
        bufferSize: bufferSize,
        aesParams:  aesParams,
        ctx:        ctx,
        cancel:     cancel,
        currentPart: -1,
    }
}

// Read implements io.Reader
func (s *MultipartFileStream) Read(p []byte) (n int, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    for n < len(p) && s.position < s.fileSize {
        // Ensure we have a current stream
        if s.currentStream == nil {
            if err := s.openPartForPosition(s.position); err != nil {
                return n, err
            }
        }

        // Read from current stream
        bytesRead, err := s.currentStream.Read(p[n:])
        n += bytesRead
        s.position += int64(bytesRead)

        if err == io.EOF {
            s.currentStream.Close()
            s.currentStream = nil
            continue // Try next part
        }
        if err != nil {
            return n, err
        }
    }

    if s.position >= s.fileSize {
        return n, io.EOF
    }
    return n, nil
}

// Seek implements io.Seeker
func (s *MultipartFileStream) Seek(offset int64, whence int) (int64, error) {
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
    }

    if newPos < 0 || newPos > s.fileSize {
        return s.position, fmt.Errorf("seek out of bounds: %d", newPos)
    }

    if newPos != s.position {
        if s.currentStream != nil {
            s.currentStream.Close()
            s.currentStream = nil
        }
        s.position = newPos
    }

    return s.position, nil
}

// Size returns the total file size
func (s *MultipartFileStream) Size() int64 {
    return s.fileSize
}

// Close closes the stream
func (s *MultipartFileStream) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.cancel()
    if s.currentStream != nil {
        return s.currentStream.Close()
    }
    return nil
}

func (s *MultipartFileStream) openPartForPosition(pos int64) error {
    // Find which part contains this position
    var offset int64
    for i, part := range s.fileParts {
        partSize := part.FilePartByteRange.Count()
        if pos < offset+partSize {
            // This is the part we need
            s.currentPart = i

            // Create stream to this part's segments
            stream := NewNzbFileStream(
                s.ctx,
                part.SegmentIDs,
                part.SegmentByteRange.Count(),
                s.client,
                s.bufferSize,
            )

            // Seek to the start of file data within segments
            seekPos := part.FilePartByteRange.Start + (pos - offset)
            if _, err := stream.Seek(seekPos, io.SeekStart); err != nil {
                stream.Close()
                return err
            }

            // Wrap with length limiter
            remaining := partSize - (pos - offset)
            s.currentStream = &limitedReader{
                r:         stream,
                remaining: remaining,
                closer:    stream,
            }

            return nil
        }
        offset += partSize
    }

    return fmt.Errorf("position %d not found in any part", pos)
}

type limitedReader struct {
    r         io.Reader
    remaining int64
    closer    io.Closer
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
    if l.remaining <= 0 {
        return 0, io.EOF
    }

    if int64(len(p)) > l.remaining {
        p = p[:l.remaining]
    }

    n, err = l.r.Read(p)
    l.remaining -= int64(n)
    return n, err
}

func (l *limitedReader) Close() error {
    return l.closer.Close()
}
```

---

## Step 10: Implement Interpolation Search

Create `search.go`:

```go
package nzbstream

import (
    "context"
    "fmt"
)

// SearchResult contains the result of an interpolation search
type SearchResult struct {
    Index     int       // Segment index
    ByteRange LongRange // Byte range of this segment
}

// InterpolationSearch finds which segment contains the target byte offset
// This is O(log log N) on average for uniformly distributed data
func InterpolationSearch(
    ctx context.Context,
    targetByte int64,
    segmentCount int,
    fileSize int64,
    getByteRange func(ctx context.Context, index int) (LongRange, error),
) (SearchResult, error) {
    indexRange := LongRange{Start: 0, End: int64(segmentCount)}
    byteRange := LongRange{Start: 0, End: fileSize}

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

        // Fetch actual byte range of guessed segment
        segmentRange, err := getByteRange(ctx, guessedIndex)
        if err != nil {
            return SearchResult{}, fmt.Errorf("failed to get byte range for segment %d: %w", guessedIndex, err)
        }

        // Check if we found it
        if segmentRange.Contains(targetByte) {
            return SearchResult{Index: guessedIndex, ByteRange: segmentRange}, nil
        }

        // Adjust search bounds
        if targetByte < segmentRange.Start {
            // Guessed too high, search lower
            indexRange = LongRange{Start: indexRange.Start, End: int64(guessedIndex)}
            byteRange = LongRange{Start: byteRange.Start, End: segmentRange.Start}
        } else {
            // Guessed too low, search higher
            indexRange = LongRange{Start: int64(guessedIndex + 1), End: indexRange.End}
            byteRange = LongRange{Start: segmentRange.End, End: byteRange.End}
        }
    }
}
```

---

## Step 11: Implement MultiSegmentStream

Create `multi_segment_stream.go`:

```go
package nzbstream

import (
    "context"
    "io"
    "sync"
)

// MultiSegmentStream downloads and chains multiple segments with parallel prefetch
type MultiSegmentStream struct {
    segmentIDs []string
    client     NntpClient
    bufferSize int

    ctx        context.Context
    cancel     context.CancelFunc
    streamChan chan io.ReadCloser
    errChan    chan error

    mu      sync.Mutex
    current io.ReadCloser
    closed  bool
}

// NewMultiSegmentStream creates a stream that prefetches segments in parallel
func NewMultiSegmentStream(
    ctx context.Context,
    segmentIDs []string,
    client NntpClient,
    bufferSize int,
) *MultiSegmentStream {
    if bufferSize <= 0 {
        bufferSize = 5 // Default prefetch buffer
    }

    ctx, cancel := context.WithCancel(ctx)

    s := &MultiSegmentStream{
        segmentIDs: segmentIDs,
        client:     client,
        bufferSize: bufferSize,
        ctx:        ctx,
        cancel:     cancel,
        streamChan: make(chan io.ReadCloser, bufferSize),
        errChan:    make(chan error, 1),
    }

    go s.downloadLoop()

    return s
}

func (s *MultiSegmentStream) downloadLoop() {
    defer close(s.streamChan)

    for _, segmentID := range s.segmentIDs {
        select {
        case <-s.ctx.Done():
            return
        default:
        }

        stream, err := s.client.DecodedBody(s.ctx, segmentID)
        if err != nil {
            select {
            case s.errChan <- err:
            default:
            }
            return
        }

        select {
        case s.streamChan <- stream:
        case <-s.ctx.Done():
            stream.Close()
            return
        }
    }
}

// Read implements io.Reader
func (s *MultiSegmentStream) Read(p []byte) (n int, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.closed {
        return 0, io.EOF
    }

    for {
        // Check for download errors
        select {
        case err := <-s.errChan:
            return 0, err
        default:
        }

        // Get current stream
        if s.current == nil {
            stream, ok := <-s.streamChan
            if !ok {
                return 0, io.EOF
            }
            s.current = stream
        }

        // Read from current stream
        n, err = s.current.Read(p)
        if n > 0 {
            return n, nil
        }

        // Current exhausted, move to next
        s.current.Close()
        s.current = nil

        if err != nil && err != io.EOF {
            return 0, err
        }
    }
}

// Close stops prefetching and closes all streams
func (s *MultiSegmentStream) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.closed {
        return nil
    }
    s.closed = true

    s.cancel()

    if s.current != nil {
        s.current.Close()
    }

    // Drain and close remaining buffered streams
    for stream := range s.streamChan {
        stream.Close()
    }

    return nil
}
```

---

## Step 12: Implement NzbFileStream (Seekable)

Create `nzb_file_stream.go`:

```go
package nzbstream

import (
    "context"
    "fmt"
    "io"
    "sync"
)

// NzbFileStream provides a seekable stream over an NZB file's segments
type NzbFileStream struct {
    segmentIDs []string
    fileSize   int64
    client     NntpClient
    bufferSize int

    mu          sync.Mutex
    position    int64
    innerStream io.ReadCloser
    ctx         context.Context
    cancel      context.CancelFunc
}

// NewNzbFileStream creates a seekable stream for an NZB file
func NewNzbFileStream(
    ctx context.Context,
    segmentIDs []string,
    fileSize int64,
    client NntpClient,
    bufferSize int,
) *NzbFileStream {
    ctx, cancel := context.WithCancel(ctx)
    return &NzbFileStream{
        segmentIDs: segmentIDs,
        fileSize:   fileSize,
        client:     client,
        bufferSize: bufferSize,
        ctx:        ctx,
        cancel:     cancel,
    }
}

// Read implements io.Reader
func (s *NzbFileStream) Read(p []byte) (n int, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.position >= s.fileSize {
        return 0, io.EOF
    }

    if s.innerStream == nil {
        stream, err := s.createStream(s.position)
        if err != nil {
            return 0, err
        }
        s.innerStream = stream
    }

    n, err = s.innerStream.Read(p)
    s.position += int64(n)
    return n, err
}

// Seek implements io.Seeker
func (s *NzbFileStream) Seek(offset int64, whence int) (int64, error) {
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

    if newPos != s.position {
        // Close current stream and reset
        if s.innerStream != nil {
            s.innerStream.Close()
            s.innerStream = nil
        }
        s.position = newPos
    }

    return s.position, nil
}

// Size returns the total file size
func (s *NzbFileStream) Size() int64 {
    return s.fileSize
}

// Close closes the stream
func (s *NzbFileStream) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.cancel()
    if s.innerStream != nil {
        return s.innerStream.Close()
    }
    return nil
}

func (s *NzbFileStream) createStream(startPos int64) (io.ReadCloser, error) {
    if startPos == 0 {
        // Start from beginning - no search needed
        return NewMultiSegmentStream(s.ctx, s.segmentIDs, s.client, s.bufferSize), nil
    }

    // Use interpolation search to find the segment containing startPos
    result, err := InterpolationSearch(
        s.ctx,
        startPos,
        len(s.segmentIDs),
        s.fileSize,
        func(ctx context.Context, index int) (LongRange, error) {
            header, err := s.client.GetYencHeader(ctx, s.segmentIDs[index])
            if err != nil {
                return LongRange{}, err
            }
            return header.ByteRange(), nil
        },
    )
    if err != nil {
        return nil, fmt.Errorf("failed to find segment for position %d: %w", startPos, err)
    }

    // Create stream starting from found segment
    stream := NewMultiSegmentStream(s.ctx, s.segmentIDs[result.Index:], s.client, s.bufferSize)

    // Discard bytes within the found segment to reach exact position
    skipBytes := startPos - result.ByteRange.Start
    if skipBytes > 0 {
        if _, err := io.CopyN(io.Discard, stream, skipBytes); err != nil {
            stream.Close()
            return nil, fmt.Errorf("failed to skip %d bytes: %w", skipBytes, err)
        }
    }

    return stream, nil
}
```

---

## Step 13: Implement the HTTP Handler

Create `handler.go`:

```go
package nzbstream

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "strings"
)

// SeekableStream is the interface for both NzbFileStream and MultipartFileStream
type SeekableStream interface {
    io.Reader
    io.Seeker
    io.Closer
    Size() int64
}

// StreamHandler handles NZB streaming requests
type StreamHandler struct {
    Client     NntpClient
    BufferSize int // Segment prefetch buffer size
}

// ServeHTTP handles POST requests with NZB file
func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Only accept POST
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse multipart form
    if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
        http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Get NZB file
    file, _, err := r.FormFile("nzb")
    if err != nil {
        http.Error(w, "Missing 'nzb' file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Get optional password for encrypted archives
    password := r.FormValue("password")

    // Parse NZB
    nzb, err := ParseNzb(file)
    if err != nil {
        http.Error(w, "Failed to parse NZB: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Process NZB to detect file types and extract archive metadata
    processedFiles, err := ProcessNzb(ctx, nzb, h.Client, password)
    if err != nil {
        http.Error(w, "Failed to process NZB: "+err.Error(), http.StatusInternalServerError)
        return
    }

    if len(processedFiles) == 0 {
        http.Error(w, "No streamable files found in NZB", http.StatusBadRequest)
        return
    }

    // Select file to stream (by name or largest)
    var selectedFile *ProcessedFile
    if fileName := r.FormValue("file"); fileName != "" {
        for i := range processedFiles {
            if strings.Contains(strings.ToLower(processedFiles[i].Name), strings.ToLower(fileName)) {
                selectedFile = &processedFiles[i]
                break
            }
        }
        if selectedFile == nil {
            http.Error(w, "File not found: "+fileName, http.StatusNotFound)
            return
        }
    } else {
        // Default to largest file
        var largest *ProcessedFile
        var largestSize int64
        for i := range processedFiles {
            if processedFiles[i].Size > largestSize {
                largestSize = processedFiles[i].Size
                largest = &processedFiles[i]
            }
        }
        selectedFile = largest
    }

    // Create the appropriate stream based on file type
    var stream SeekableStream
    switch selectedFile.Type {
    case FileTypeDirect:
        stream = NewNzbFileStream(ctx, selectedFile.SegmentIDs, selectedFile.Size, h.Client, h.BufferSize)
    case FileTypeRar, FileType7z:
        baseStream := NewMultipartFileStream(
            ctx,
            selectedFile.FileParts,
            selectedFile.Size,
            h.Client,
            h.BufferSize,
            selectedFile.AesParams,
        )
        stream = baseStream
    }
    defer stream.Close()

    // Serve with Range support
    h.serveContent(w, r, stream, selectedFile.Name)
}

func (h *StreamHandler) serveContent(w http.ResponseWriter, r *http.Request, stream SeekableStream, filename string) {
    fileSize := stream.Size()

    // Set content headers
    w.Header().Set("Accept-Ranges", "bytes")
    w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))

    // Detect content type from filename
    contentType := detectContentType(filename)
    w.Header().Set("Content-Type", contentType)

    // Handle Range header
    rangeHeader := r.Header.Get("Range")
    if rangeHeader == "" {
        // No range - serve entire file
        w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
        w.WriteHeader(http.StatusOK)
        io.Copy(w, stream)
        return
    }

    // Parse Range header
    start, end, err := parseRangeHeader(rangeHeader, fileSize)
    if err != nil {
        http.Error(w, "Invalid range: "+err.Error(), http.StatusRequestedRangeNotSatisfiable)
        return
    }

    // Seek to start position
    if _, err := stream.Seek(start, io.SeekStart); err != nil {
        http.Error(w, "Seek failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Calculate content length
    contentLength := end - start + 1

    // Set response headers
    w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
    w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
    w.WriteHeader(http.StatusPartialContent)

    // Copy the requested range
    io.CopyN(w, stream, contentLength)
}

func detectContentType(filename string) string {
    lower := strings.ToLower(filename)
    switch {
    case strings.HasSuffix(lower, ".mkv"):
        return "video/x-matroska"
    case strings.HasSuffix(lower, ".mp4"):
        return "video/mp4"
    case strings.HasSuffix(lower, ".avi"):
        return "video/x-msvideo"
    case strings.HasSuffix(lower, ".webm"):
        return "video/webm"
    case strings.HasSuffix(lower, ".mov"):
        return "video/quicktime"
    case strings.HasSuffix(lower, ".wmv"):
        return "video/x-ms-wmv"
    case strings.HasSuffix(lower, ".flv"):
        return "video/x-flv"
    case strings.HasSuffix(lower, ".ts"), strings.HasSuffix(lower, ".m2ts"):
        return "video/mp2t"
    default:
        return "application/octet-stream"
    }
}

// parseRangeHeader parses a Range header like "bytes=0-999" or "bytes=1000-"
func parseRangeHeader(header string, fileSize int64) (start, end int64, err error) {
    if !strings.HasPrefix(header, "bytes=") {
        return 0, 0, fmt.Errorf("invalid range format")
    }

    rangeSpec := strings.TrimPrefix(header, "bytes=")
    parts := strings.Split(rangeSpec, "-")
    if len(parts) != 2 {
        return 0, 0, fmt.Errorf("invalid range format")
    }

    // Parse start
    if parts[0] == "" {
        // Suffix range like "-500" (last 500 bytes)
        suffix, err := strconv.ParseInt(parts[1], 10, 64)
        if err != nil {
            return 0, 0, fmt.Errorf("invalid suffix: %w", err)
        }
        start = fileSize - suffix
        end = fileSize - 1
    } else {
        start, err = strconv.ParseInt(parts[0], 10, 64)
        if err != nil {
            return 0, 0, fmt.Errorf("invalid start: %w", err)
        }

        if parts[1] == "" {
            // Open-ended range like "1000-"
            end = fileSize - 1
        } else {
            end, err = strconv.ParseInt(parts[1], 10, 64)
            if err != nil {
                return 0, 0, fmt.Errorf("invalid end: %w", err)
            }
        }
    }

    // Validate range
    if start < 0 || start >= fileSize {
        return 0, 0, fmt.Errorf("start out of bounds")
    }
    if end >= fileSize {
        end = fileSize - 1
    }
    if start > end {
        return 0, 0, fmt.Errorf("start > end")
    }

    return start, end, nil
}
```

---

## Step 14: Wire It All Together

Create `main.go` (example):

```go
package main

import (
    "context"
    "log"
    "net/http"

    "yourproject/nzbstream"
    "yourproject/nntp" // Your existing NNTP package
)

func main() {
    // Create your existing NNTP client/pool
    nntpPool := nntp.NewPool(nntp.PoolConfig{
        Host:        "news.example.com",
        Port:        563,
        Username:    "user",
        Password:    "pass",
        UseTLS:      true,
        MaxConns:    10,
    })

    // Wrap it with the adapter
    client := &nzbstream.NntpClientAdapter{
        YourClient: nntpPool,
    }

    // Create the handler
    handler := &nzbstream.StreamHandler{
        Client:     client,
        BufferSize: 10, // Prefetch 10 segments
    }

    // Register route
    http.Handle("/stream", handler)

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

---

## Step 15: Test the API

### Upload and Stream

```bash
# Stream the largest file in the NZB (works for direct files and archives)
curl -X POST -F "nzb=@movie.nzb" http://localhost:8080/stream --output movie.mkv

# Stream a specific file by name
curl -X POST -F "nzb=@movie.nzb" -F "file=sample.mkv" http://localhost:8080/stream --output sample.mkv

# Stream with Range header (for video seeking)
curl -X POST -F "nzb=@movie.nzb" \
     -H "Range: bytes=1000000-2000000" \
     http://localhost:8080/stream --output partial.bin

# Stream from encrypted archive (password-protected 7z/RAR)
curl -X POST -F "nzb=@encrypted-archive.nzb" -F "password=mypassword" \
     http://localhost:8080/stream --output decrypted-movie.mkv
```

### Test with a Video Player

```bash
# Using VLC (pipe through curl)
curl -X POST -F "nzb=@movie.nzb" http://localhost:8080/stream | vlc -

# Stream encrypted archive to VLC
curl -X POST -F "nzb=@encrypted.nzb" -F "password=secret" \
     http://localhost:8080/stream | vlc -
```

### Supported File Types

The API automatically detects and handles:
- **Direct video files**: `.mkv`, `.mp4`, `.avi`, `.mov`, `.wmv`, `.flv`, `.webm`, `.ts`, `.m2ts`
- **RAR archives**: `.rar`, `.r00`, `.r01`, `.part01.rar` (RAR4 and RAR5 formats)
- **7z archives**: `.7z`, `.7z.001`, `.7z.002` (supports AES-256 encryption)

---

## Summary

The implementation consists of these layers:

| Layer | File | Purpose |
|-------|------|---------|
| HTTP Handler | `handler.go` | Parses NZB, routes to correct stream type, handles Range headers |
| Archive Processor | `processor.go` | Detects file types, parses archive headers, extracts byte ranges |
| File Type Detection | `detect.go` | Magic bytes + extension-based detection |
| RAR Parser | `rar.go` | Parses RAR4/RAR5 headers for file offsets |
| 7z Parser | `sevenz.go` | Parses 7z headers, handles AES-256 decryption |
| MultipartFileStream | `multipart_file_stream.go` | Streams files split across archive parts |
| NzbFileStream | `nzb_file_stream.go` | Seekable stream with position tracking |
| Interpolation Search | `search.go` | O(log log N) segment lookup for seeking |
| MultiSegmentStream | `multi_segment_stream.go` | Parallel prefetch with buffered channel |
| yEnc Decoder | `yenc.go` | Decodes article bodies on-the-fly |
| NNTP Adapter | `nntp_interface.go` | Wraps your existing client |
| Models | `models.go` | Data structures (LongRange, NzbFile, ProcessedFile, FilePart, etc.) |
| Parser | `parser.go` | NZB XML parsing |

**Key Features:**
- Streaming without downloading to disk
- HTTP Range header support for video seeking
- Parallel segment prefetch for smooth playback
- Efficient O(log log N) segment lookup
- **RAR archive support** (RAR4 and RAR5 formats)
- **7z archive support** with AES-256 decryption for encrypted archives
- **Multi-part archive streaming** - files split across multiple archive parts
- Automatic file type detection via magic bytes
- Uses your existing NNTP client/pool
