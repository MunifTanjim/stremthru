# NzbDAV Usenet Streaming - Go Implementation Guide

This document provides the core implementation details needed to implement the usenet file streaming functionality in Go.

---

## 0. End-to-End Flow

### How NZBs Enter the System

Before streaming can happen, NZBs must be added to the system and processed. NzbDAV exposes a SABnzbd-compatible API that *arr apps (Radarr, Sonarr, etc.) use to submit downloads.

#### Entry Points

| Endpoint | Description |
|----------|-------------|
| `POST /api?mode=addfile` | Upload NZB file directly |
| `POST /api?mode=addurl` | Provide URL to fetch NZB from |

#### Complete Flow: From NZB Submission to Streaming

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        NZB INGESTION & PROCESSING                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   *arr App (Radarr/Sonarr/Lidarr/etc)                                       │
│           │                                                                  │
│           │ POST /api?mode=addfile  (SABnzbd-compatible)                    │
│           ▼                                                                  │
│   ┌─────────────────────┐                                                    │
│   │  AddFileController  │ ◄── Parse NZB XML, extract metadata               │
│   └──────────┬──────────┘                                                    │
│              │                                                               │
│              ▼                                                               │
│   ┌─────────────────────┐                                                    │
│   │      Database       │ ◄── Store QueueItem + QueueNzbContents            │
│   └──────────┬──────────┘                                                    │
│              │                                                               │
│              │ QueueManager.AwakenQueue()                                    │
│              ▼                                                               │
│   ┌─────────────────────┐                                                    │
│   │    QueueManager     │ ◄── Background loop, processes one item at a time │
│   └──────────┬──────────┘                                                    │
│              │                                                               │
│              ▼                                                               │
│   ┌─────────────────────┐                                                    │
│   │  QueueItemProcessor │ ◄── Fetch first segments, detect types,           │
│   └──────────┬──────────┘     parse archives, save file metadata            │
│              │                                                               │
│              ▼                                                               │
│      DavNzbFile (direct)                                                     │
│            or                                                                │
│      DavMultipartFile (archived)                                             │
│              │                                                               │
│              ▼                                                               │
│      Ready for WebDAV streaming                                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Go Implementation

```go
// Database models for the queue
type QueueItem struct {
    ID                 uuid.UUID
    CreatedAt          time.Time
    FileName           string    // Original NZB filename
    JobName            string    // Display name (filename without .nzb)
    Category           string    // e.g., "movies", "tv"
    Priority           int       // Processing priority
    TotalSegmentBytes  int64     // Sum of all segment sizes
    PauseUntil         *time.Time
}

type QueueNzbContents struct {
    ID          uuid.UUID  // Same as QueueItem.ID
    NzbContents string     // Raw NZB XML
}
```

```go
// AddFile endpoint handler
func (h *AddFileHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form to get NZB file
    file, header, _ := r.FormFile("nzbfile")
    nzbContents, _ := io.ReadAll(file)

    // Parse NZB to extract metadata
    nzb, _ := ParseNzb(nzbContents)

    // Create queue item
    queueItem := QueueItem{
        ID:                uuid.New(),
        CreatedAt:         time.Now(),
        FileName:          header.Filename,
        JobName:           strings.TrimSuffix(header.Filename, ".nzb"),
        Category:          r.FormValue("cat"),
        TotalSegmentBytes: nzb.TotalSegmentBytes(),
    }

    // Save to database
    db.Save(&queueItem)
    db.Save(&QueueNzbContents{ID: queueItem.ID, NzbContents: string(nzbContents)})

    // Wake up the queue processor
    queueManager.AwakenQueue()

    // Return SABnzbd-compatible response
    json.NewEncoder(w).Encode(map[string]any{
        "status":  true,
        "nzo_ids": []string{queueItem.ID.String()},
    })
}
```

```go
// QueueManager runs in the background
type QueueManager struct {
    db           *Database
    usenetClient NntpClient
    wakeChan     chan struct{}
}

func (m *QueueManager) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-m.wakeChan:
            // Awakened by new item
        case <-time.After(time.Minute):
            // Periodic check
        }

        // Get next queue item
        item, nzbContents := m.db.GetTopQueueItem()
        if item == nil {
            continue
        }

        // Process it
        processor := NewQueueItemProcessor(item, nzbContents, m.usenetClient)
        processor.Process(ctx)
    }
}

func (m *QueueManager) AwakenQueue() {
    select {
    case m.wakeChan <- struct{}{}:
    default:
        // Already awake
    }
}
```

---

### Streaming Flow

Once NZBs are processed, here's how streaming works when a client requests a file:

#### Visual Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  WebDAV GET /movie.mkv (Range: bytes=1000000-)                              │
│         │                                                                   │
│         ▼                                                                   │
│  ┌─────────────────┐                                                        │
│  │  NzbFileStream  │ ◄── Seekable stream wrapping ordered segment IDs      │
│  │   (Section 6)   │     Position 0? → Direct read                         │
│  └────────┬────────┘     Position N? → InterpolationSearch finds segment   │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────┐                                                    │
│  │ MultiSegmentStream  │ ◄── Prefetches segments in background goroutine   │
│  │     (Section 6)     │     Buffered channel holds decoded streams ahead  │
│  └──────────┬──────────┘                                                    │
│             │                                                               │
│             ▼                                                               │
│  ┌──────────────────────────────────────────────────────────────┐          │
│  │                    NNTP Client Chain                          │          │
│  │  ConnectionPool ──▶ MultiProviderClient ──▶ BaseNntpClient   │          │
│  │   (Section 7)         (Section 8)            (Section 3)     │          │
│  └────────────────────────────┬─────────────────────────────────┘          │
│                               │                                             │
│                               ▼                                             │
│                       Usenet Server                                         │
│                      BODY <message-id>                                      │
│                               │                                             │
│                               ▼                                             │
│                     ┌──────────────────┐                                    │
│                     │   yEnc Decoder   │ ◄── Decodes binary from text      │
│                     │   (Section 4)    │                                    │
│                     └────────┬─────────┘                                    │
│                              │                                              │
│                              ▼                                              │
│                      Raw bytes → Client                                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Step-by-Step: What Happens on a Request

1. **Client requests file** via WebDAV GET, possibly with a byte range (e.g., seeking in a video)

2. **NzbFileStream** receives the request with target byte position:
   - If position = 0: Skip to step 4
   - If position > 0: Use **InterpolationSearch** (Section 5) to find which segment contains that byte

3. **InterpolationSearch** efficiently locates the segment:
   - Estimates segment index based on average bytes-per-segment
   - Fetches yEnc headers from that segment to get exact byte range
   - Adjusts search bounds and repeats until found
   - Returns segment index + byte offset within that segment

4. **MultiSegmentStream** starts from the found segment:
   - Spawns background goroutine to download segments ahead
   - Fills buffered channel with decoded streams (prefetch buffer)
   - Reader pulls from channel, seamlessly chaining segments

5. **For each segment**, the NNTP client chain:
   - **ConnectionPool** provides an idle or new connection
   - **MultiProviderClient** tries primary provider first, fails over to backups on 430 errors
   - **BaseNntpClient** sends `BODY <message-id>` command

6. **yEnc Decoder** wraps the NNTP response:
   - Parses `=ybegin`/`=ypart` headers for byte range info
   - Decodes escape sequences on-the-fly
   - Returns raw binary as `io.Reader`

7. **Bytes flow to client** as they're decoded - no local storage needed

### Key Behaviors

| Scenario | What Happens |
|----------|--------------|
| **Sequential read** | Direct pipeline: MultiSegmentStream prefetches ahead while reader consumes |
| **Random seek** | InterpolationSearch finds segment in O(log log N), discards offset bytes within segment |
| **Segment missing** | MultiProviderClient retries with backup providers before failing |
| **Slow provider** | ConnectionPool's semaphore limits concurrent connections; prioritized scheduling favors streaming over health checks |
| **Connection idle** | Pool's sweeper closes connections unused for 30+ seconds |

### Archive Files (RAR/7z)

For files inside archives, an additional layer handles multi-part assembly:

```
┌─────────────────────────────────────────┐
│  DavMultipartFileStream / MultipartFile │ ◄── Represents file split across parts
└──────────────────┬──────────────────────┘
                   │
    Uses InterpolationSearch to find which part
                   │
                   ▼
    Creates NzbFileStream for that part, seeks to offset within part
```

Each `FilePart` knows:
- Which segment IDs make up that archive part
- The byte range of segments (`SegmentIDByteRange`)
- Where the actual file data sits within those bytes (`FilePartByteRange`)

For encrypted 7z archives, `AesDecoderStream` (Section 9) wraps the stream with seekable AES-CBC decryption.

### NZB Processing: Detection and Routing

Before streaming can happen, the NZB must be processed to detect file types and extract metadata. Here's how:

#### Step 1: Fetch First 16KB of Each File

```go
type NzbFileWithFirstSegment struct {
    NzbFile            NzbFile
    First16KB          []byte    // First 16KB of decoded content
    Header             *YencHeader
    MissingFirstSegment bool
    ReleaseDate        time.Time
}

// Magic byte signatures for archive detection
var (
    Rar4Magic = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}       // "Rar!..."
    Rar5Magic = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00} // "Rar!...." (v5)
    SevenZMagic = []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}           // "7z...."
)

func (f *NzbFileWithFirstSegment) HasRarMagic() bool {
    return bytes.HasPrefix(f.First16KB, Rar4Magic) ||
           bytes.HasPrefix(f.First16KB, Rar5Magic)
}
```

#### Step 2: Classify Each File

```go
func ClassifyFile(info FileInfo) string {
    // Check by magic bytes first (most reliable)
    if info.HasRarMagic {
        return "rar"
    }

    // Fall back to extension-based detection
    filename := strings.ToLower(info.FileName)

    switch {
    case regexp.MustCompile(`\.7z(\.\d+)?$`).MatchString(filename):
        return "7z"
    case strings.HasSuffix(filename, ".rar") ||
         regexp.MustCompile(`\.r\d+$`).MatchString(filename):
        return "rar"
    case regexp.MustCompile(`\.mkv\.\d+$`).MatchString(filename):
        return "multipart-mkv"
    default:
        return "direct"  // .mkv, .mp4, etc.
    }
}
```

#### Step 3: Process by Type

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           NZB Processing Pipeline                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Parse NZB → Fetch first 16KB of each file → Check magic + extension       │
│                                    │                                         │
│              ┌─────────────────────┼─────────────────────┐                  │
│              │                     │                     │                  │
│              ▼                     ▼                     ▼                  │
│     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐         │
│     │  Direct File   │    │  RAR Archive   │    │  7z Archive    │         │
│     │  (.mkv, .mp4)  │    │  (magic/ext)   │    │  (.7z ext)     │         │
│     └───────┬────────┘    └───────┬────────┘    └───────┬────────┘         │
│             │                     │                     │                   │
│             ▼                     ▼                     ▼                   │
│     ┌────────────────┐    ┌────────────────┐    ┌────────────────┐         │
│     │ FileProcessor  │    │ RarProcessor   │    │SevenZipProcessor│        │
│     │ Record segment │    │ Parse headers, │    │ Parse headers,  │        │
│     │ IDs + size     │    │ extract byte   │    │ extract byte    │        │
│     │                │    │ ranges per file│    │ ranges per file │        │
│     └───────┬────────┘    └───────┬────────┘    └───────┬────────┘         │
│             │                     │                     │                   │
│             ▼                     ▼                     ▼                   │
│     ┌────────────────┐    ┌─────────────────────────────┴──┐               │
│     │  DavNzbFile    │    │       DavMultipartFile         │               │
│     │  { SegmentIDs }│    │ { FileParts[], AesParams? }    │               │
│     └────────────────┘    └────────────────────────────────┘               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Database Models

```go
// For direct files (video directly in NZB)
type DavNzbFile struct {
    ID         uuid.UUID
    SegmentIDs []string  // Ordered segment message IDs
}

// For files inside archives (RAR/7z split across parts)
type DavMultipartFile struct {
    ID       uuid.UUID
    Metadata MultipartMetadata
}

type MultipartMetadata struct {
    AesParams *AesParams  // nil if not encrypted
    FileParts []FilePart  // Ordered by part number
}

type FilePart struct {
    SegmentIDs         []string  // Segment IDs for this archive part
    SegmentIDByteRange LongRange // Total byte range of segments
    FilePartByteRange  LongRange // Where file data sits within segments
}
```

#### Archive Processor: Extracting File Metadata

The archive processors don't decompress anything - they just parse headers to find byte offsets:

```go
// RarProcessor: Stream archive, parse headers, record byte ranges
func (p *RarProcessor) Process(ctx context.Context) (*RarResult, error) {
    // Open seekable stream to the RAR file
    stream := p.client.GetFileStream(p.nzbFile, p.fileSize, 0)
    defer stream.Close()

    // Parse RAR headers (reads header blocks, not file data)
    headers, err := ParseRarHeaders(stream, p.password)
    if err != nil {
        return nil, err
    }

    var segments []StoredFileSegment
    for _, header := range headers {
        if header.Type != FileHeader {
            continue
        }

        segments = append(segments, StoredFileSegment{
            NzbFile:          p.nzbFile,
            PartSize:         stream.Length(),
            ArchiveName:      p.getArchiveName(),
            PartNumber:       p.getPartNumber(headers),
            PathWithinArchive: header.FileName,
            ByteRangeWithinPart: LongRange{
                Start: header.DataStartPosition,
                End:   header.DataStartPosition + header.CompressedSize,
            },
            AesParams: header.GetAesParams(p.password),
        })
    }

    return &RarResult{Segments: segments}, nil
}
```

#### Aggregator: Assembling Multi-Part Files

After processing, the aggregator groups parts by filename and orders by part number:

```go
func (a *RarAggregator) Aggregate(results []RarResult) []DavMultipartFile {
    // Group segments by path within archive
    byPath := make(map[string][]StoredFileSegment)
    for _, result := range results {
        for _, seg := range result.Segments {
            byPath[seg.PathWithinArchive] = append(
                byPath[seg.PathWithinArchive], seg)
        }
    }

    var files []DavMultipartFile
    for path, segments := range byPath {
        // Sort by part number
        sort.Slice(segments, func(i, j int) bool {
            return segments[i].PartNumber < segments[j].PartNumber
        })

        // Build file parts
        var parts []FilePart
        for _, seg := range segments {
            parts = append(parts, FilePart{
                SegmentIDs:         seg.NzbFile.GetSegmentIDs(),
                SegmentIDByteRange: LongRange{0, seg.PartSize},
                FilePartByteRange:  seg.ByteRangeWithinPart,
            })
        }

        files = append(files, DavMultipartFile{
            ID: uuid.New(),
            Metadata: MultipartMetadata{
                AesParams: segments[0].AesParams,
                FileParts: parts,
            },
        })
    }

    return files
}
```

#### Streaming: Choosing the Right Stream Type

When serving a file via WebDAV, the stream type is chosen based on the database model:

```go
func GetFileStream(item DavItem, client NntpClient) (io.ReadSeekCloser, error) {
    switch item.Type {
    case ItemTypeNzbFile:
        // Direct file: simple segment stream
        nzbFile, _ := db.GetNzbFile(item.ID)
        return NewNzbFileStream(nzbFile.SegmentIDs, item.FileSize, client), nil

    case ItemTypeMultipartFile:
        // Archived file: multi-part stream with byte range mapping
        multipart, _ := db.GetMultipartFile(item.ID)
        stream := NewMultipartFileStream(multipart.Metadata.FileParts, client)

        // Wrap with AES decryption if encrypted
        if multipart.Metadata.AesParams != nil {
            return NewAesDecoderStream(stream, *multipart.Metadata.AesParams), nil
        }
        return stream, nil

    default:
        return nil, errors.New("unknown item type")
    }
}
```

---

## 1. Data Structures

```go
package nzbdav

// LongRange represents a half-open byte interval [Start, End)
type LongRange struct {
    Start int64 // First byte included
    End   int64 // First byte NOT included
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

func NewLongRangeFromSize(start, size int64) LongRange {
    return LongRange{Start: start, End: start + size}
}
```

```go
// NzbFile represents a file entry from an NZB XML file
type NzbFile struct {
    Subject  string
    Poster   string
    Groups   []string
    Segments []Segment
}

// Segment represents a single article segment
type Segment struct {
    Number    int
    Bytes     int64
    MessageID string // e.g., "abc123@news.example.com"
}

// GetSegmentIDs returns just the message IDs in order
func (f *NzbFile) GetSegmentIDs() []string {
    ids := make([]string, len(f.Segments))
    for i, seg := range f.Segments {
        ids[i] = seg.MessageID
    }
    return ids
}
```

```go
// DavNzbFile is the stored representation for streaming
type DavNzbFile struct {
    ID         uuid.UUID
    SegmentIDs []string
}

// MultipartFile represents a file split across archive parts
type MultipartFile struct {
    FileParts []FilePart
}

func (m *MultipartFile) FileSize() int64 {
    if len(m.FileParts) == 0 {
        return 0
    }
    return m.FileParts[len(m.FileParts)-1].ByteRange.End
}

type FilePart struct {
    SegmentIDs         []string
    SegmentIDByteRange LongRange // Byte range covered by segments
    FilePartByteRange  LongRange // Actual file content within segments
}

// AesParams for encrypted 7z archives
type AesParams struct {
    DecodedSize int64
    IV          [16]byte
    Key         [32]byte
}
```

---

## 2. Go Libraries

### NNTP Client
```go
import (
    "github.com/Tensai75/nntp"      // or github.com/dustin/go-nntp
    "github.com/Tensai75/nntpPool"  // Connection pooling
)
```

**Recommended:** [Tensai75/nntpPool](https://pkg.go.dev/github.com/Tensai75/nntpPool) - Connection pool with SSL support

**Alternatives:**
- [dustin/go-nntp](https://github.com/dustin/go-nntp) - NNTP client/server
- [chrisfarms/nntp](https://github.com/chrisfarms/nntp) - Simple client

### yEnc Decoding
```go
import "github.com/chrisfarms/yenc"
// or
import "github.com/GJRTimmer/yenc"  // Multi-core support
```

**Recommended:** [GJRTimmer/yenc](https://github.com/GJRTimmer/yenc) - Multi-core support

### Archive Parsing
```go
import (
    "github.com/nwaples/rardecode"  // RAR headers
    "github.com/bodgit/sevenzip"    // 7z archives
)
```

**RAR:** [nwaples/rardecode](https://pkg.go.dev/github.com/nwaples/rardecode) - Pure Go RAR reader
**7z:** [bodgit/sevenzip](https://github.com/bodgit/sevenzip) - Pure Go, supports encrypted archives

---

## 3. NNTP Protocol in Go

```go
package nntp

import (
    "bufio"
    "crypto/tls"
    "fmt"
    "io"
    "net"
    "net/textproto"
)

type Client struct {
    conn   net.Conn
    reader *textproto.Reader
    writer *textproto.Writer
}

func Dial(host string, port int, useTLS bool) (*Client, error) {
    addr := fmt.Sprintf("%s:%d", host, port)

    var conn net.Conn
    var err error

    if useTLS {
        conn, err = tls.Dial("tcp", addr, &tls.Config{})
    } else {
        conn, err = net.Dial("tcp", addr)
    }
    if err != nil {
        return nil, err
    }

    c := &Client{
        conn:   conn,
        reader: textproto.NewReader(bufio.NewReader(conn)),
        writer: textproto.NewWriter(bufio.NewWriter(conn)),
    }

    // Read greeting
    code, _, err := c.reader.ReadCodeLine(200)
    if err != nil && code != 201 {
        conn.Close()
        return nil, fmt.Errorf("unexpected greeting: %d", code)
    }

    return c, nil
}

func (c *Client) Authenticate(user, pass string) error {
    // Send username
    if err := c.writer.PrintfLine("AUTHINFO USER %s", user); err != nil {
        return err
    }

    code, _, err := c.reader.ReadCodeLine(381)
    if err != nil {
        return err
    }

    // Send password
    if err := c.writer.PrintfLine("AUTHINFO PASS %s", pass); err != nil {
        return err
    }

    _, _, err = c.reader.ReadCodeLine(281)
    return err
}

// Body fetches the article body (yEnc encoded)
func (c *Client) Body(messageID string) (io.ReadCloser, error) {
    if err := c.writer.PrintfLine("BODY <%s>", messageID); err != nil {
        return nil, err
    }

    code, _, err := c.reader.ReadCodeLine(222)
    if err != nil {
        if code == 430 {
            return nil, ErrArticleNotFound
        }
        return nil, err
    }

    return c.reader.DotReader(), nil
}

// Stat checks if an article exists
func (c *Client) Stat(messageID string) (bool, error) {
    if err := c.writer.PrintfLine("STAT <%s>", messageID); err != nil {
        return false, err
    }

    code, _, err := c.reader.ReadCodeLine(223)
    if code == 430 {
        return false, nil
    }
    return err == nil, err
}

var ErrArticleNotFound = fmt.Errorf("article not found")
```

### NNTP Response Codes
- `200`: Service available, posting allowed
- `201`: Service available, posting prohibited
- `221`: Article head follows
- `222`: Article body follows
- `223`: Article exists
- `281`: Authentication accepted
- `381`: Password required
- `430`: No article with that message-id
- `481`: Authentication failed

---

## 4. yEnc Decoding in Go

### Using a Library
```go
import "github.com/chrisfarms/yenc"

func DecodeArticle(r io.Reader) (*yenc.Part, error) {
    return yenc.Decode(r)
}

// Part contains:
// - Name: original filename
// - Size: total file size
// - Begin: start offset (1-based)
// - End: end offset (inclusive)
// - Body: decoded bytes as io.Reader
```

### Custom Streaming Decoder
```go
package yenc

import (
    "bufio"
    "io"
)

type Header struct {
    Name       string
    Size       int64
    Part       int
    PartBegin  int64 // 1-based
    PartEnd    int64 // inclusive
    PartOffset int64 // 0-based = PartBegin - 1
    PartSize   int64 // = PartEnd - PartBegin + 1
}

type Decoder struct {
    r          *bufio.Reader
    header     *Header
    headerRead bool
    escapeNext bool
}

func NewDecoder(r io.Reader) *Decoder {
    return &Decoder{r: bufio.NewReader(r)}
}

func (d *Decoder) Header() (*Header, error) {
    if d.headerRead {
        return d.header, nil
    }

    h := &Header{}

    // Read =ybegin line
    line, err := d.r.ReadString('\n')
    if err != nil {
        return nil, err
    }
    parseYbegin(line, h)

    // Read =ypart line if multipart
    if h.Part > 0 {
        line, err = d.r.ReadString('\n')
        if err != nil {
            return nil, err
        }
        parseYpart(line, h)
    }

    h.PartOffset = h.PartBegin - 1
    h.PartSize = h.PartEnd - h.PartBegin + 1

    d.header = h
    d.headerRead = true
    return h, nil
}

func (d *Decoder) Read(p []byte) (n int, err error) {
    if !d.headerRead {
        if _, err := d.Header(); err != nil {
            return 0, err
        }
    }

    for n < len(p) {
        b, err := d.r.ReadByte()
        if err != nil {
            return n, err
        }

        // Check for =yend (end of data)
        if b == '=' && !d.escapeNext {
            next, _ := d.r.Peek(1)
            if len(next) > 0 && next[0] == 'y' {
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
```

### yEnc Escape Sequences
Certain bytes are escaped with `=` prefix:
- `NUL (0x00)` → `=@` (0x3D 0x40)
- `TAB (0x09)` → `=I` (0x3D 0x49)
- `LF (0x0A)` → `=J` (0x3D 0x4A)
- `CR (0x0D)` → `=M` (0x3D 0x4D)
- `= (0x3D)` → `=}` (0x3D 0x7D)

Decode: `(escaped_byte - 64 - 42) & 0xFF`

---

## 5. Interpolation Search in Go

```go
package search

import (
    "context"
    "fmt"
)

type Result struct {
    Index     int
    ByteRange LongRange
}

// Find locates which segment contains the target byte offset
func Find(
    ctx context.Context,
    searchByte int64,
    indexRange LongRange, // (0, segmentCount)
    byteRange LongRange,  // (0, fileSize)
    getByteRange func(ctx context.Context, index int) (LongRange, error),
) (Result, error) {
    for {
        select {
        case <-ctx.Done():
            return Result{}, ctx.Err()
        default:
        }

        // Validate search is possible
        if !byteRange.Contains(searchByte) || indexRange.Count() <= 0 {
            return Result{}, fmt.Errorf("corrupt file: cannot find byte %d", searchByte)
        }

        // Estimate segment based on average bytes per segment
        bytesPerSegment := float64(byteRange.Count()) / float64(indexRange.Count())
        offsetFromStart := float64(searchByte - byteRange.Start)
        guessedOffset := int64(offsetFromStart / bytesPerSegment)
        guessedIndex := int(indexRange.Start + guessedOffset)

        // Fetch actual byte range
        segmentRange, err := getByteRange(ctx, guessedIndex)
        if err != nil {
            return Result{}, err
        }

        // Validate result
        if !byteRange.ContainsRange(segmentRange) {
            return Result{}, fmt.Errorf("corrupt file: segment outside range")
        }

        if searchByte < segmentRange.Start {
            // Guessed too high, search lower
            indexRange = LongRange{Start: indexRange.Start, End: int64(guessedIndex)}
            byteRange = LongRange{Start: byteRange.Start, End: segmentRange.Start}
        } else if searchByte >= segmentRange.End {
            // Guessed too low, search higher
            indexRange = LongRange{Start: int64(guessedIndex + 1), End: indexRange.End}
            byteRange = LongRange{Start: segmentRange.End, End: byteRange.End}
        } else {
            // Found it!
            return Result{Index: guessedIndex, ByteRange: segmentRange}, nil
        }
    }
}
```

### Complexity
- Average case: O(log log N) fetches
- Worst case: O(log N) fetches
- Much faster than binary search for uniform data

---

## 6. Streaming Architecture in Go

### NzbFileStream (seekable)

```go
package streams

import (
    "context"
    "fmt"
    "io"
    "sync"
)

type NzbFileStream struct {
    segmentIDs []string
    fileSize   int64
    client     NntpClient
    bufferSize int

    mu          sync.Mutex
    position    int64
    innerStream io.ReadCloser
}

func NewNzbFileStream(segmentIDs []string, fileSize int64, client NntpClient, bufferSize int) *NzbFileStream {
    return &NzbFileStream{
        segmentIDs: segmentIDs,
        fileSize:   fileSize,
        client:     client,
        bufferSize: bufferSize,
    }
}

func (s *NzbFileStream) Read(p []byte) (n int, err error) {
    return s.ReadContext(context.Background(), p)
}

func (s *NzbFileStream) ReadContext(ctx context.Context, p []byte) (n int, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.innerStream == nil {
        stream, err := s.getStream(ctx, s.position)
        if err != nil {
            return 0, err
        }
        s.innerStream = stream
    }

    n, err = s.innerStream.Read(p)
    s.position += int64(n)
    return n, err
}

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
    }

    if newPos < 0 || newPos > s.fileSize {
        return s.position, fmt.Errorf("seek out of bounds")
    }

    if newPos != s.position {
        if s.innerStream != nil {
            s.innerStream.Close()
            s.innerStream = nil
        }
        s.position = newPos
    }

    return s.position, nil
}

func (s *NzbFileStream) getStream(ctx context.Context, startPos int64) (io.ReadCloser, error) {
    if startPos == 0 {
        return NewMultiSegmentStream(ctx, s.segmentIDs, s.client, s.bufferSize), nil
    }

    // Find segment containing startPos
    result, err := Find(ctx, startPos,
        LongRange{0, int64(len(s.segmentIDs))},
        LongRange{0, s.fileSize},
        func(ctx context.Context, i int) (LongRange, error) {
            return s.client.GetYencHeaders(ctx, s.segmentIDs[i])
        },
    )
    if err != nil {
        return nil, err
    }

    stream := NewMultiSegmentStream(ctx, s.segmentIDs[result.Index:], s.client, s.bufferSize)

    // Discard bytes within the found segment
    skipBytes := startPos - result.ByteRange.Start
    if skipBytes > 0 {
        if _, err := io.CopyN(io.Discard, stream, skipBytes); err != nil {
            stream.Close()
            return nil, err
        }
    }

    return stream, nil
}

func (s *NzbFileStream) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.innerStream != nil {
        return s.innerStream.Close()
    }
    return nil
}
```

### MultiSegmentStream (parallel prefetch)

```go
package streams

import (
    "context"
    "io"
)

type MultiSegmentStream struct {
    segmentIDs []string
    client     NntpClient

    ctx        context.Context
    cancel     context.CancelFunc
    streamChan chan io.ReadCloser
    errChan    chan error

    current io.ReadCloser
}

func NewMultiSegmentStream(ctx context.Context, segmentIDs []string, client NntpClient, bufferSize int) *MultiSegmentStream {
    ctx, cancel := context.WithCancel(ctx)

    s := &MultiSegmentStream{
        segmentIDs: segmentIDs,
        client:     client,
        ctx:        ctx,
        cancel:     cancel,
        streamChan: make(chan io.ReadCloser, bufferSize),
        errChan:    make(chan error, 1),
    }

    go s.downloadSegments()

    return s
}

func (s *MultiSegmentStream) downloadSegments() {
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

func (s *MultiSegmentStream) Read(p []byte) (n int, err error) {
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

func (s *MultiSegmentStream) Close() error {
    s.cancel()

    if s.current != nil {
        s.current.Close()
    }

    // Drain and close remaining streams
    for stream := range s.streamChan {
        stream.Close()
    }

    return nil
}
```

---

## 7. Connection Pool in Go

```go
package pool

import (
    "container/list"
    "context"
    "io"
    "sync"
    "time"
)

type ConnectionPool struct {
    maxConns    int
    factory     func(context.Context) (io.Closer, error)
    idleTimeout time.Duration

    mu   sync.Mutex
    idle *list.List // of *pooledConn
    live int
    sem  chan struct{}
}

type pooledConn struct {
    conn     io.Closer
    lastUsed time.Time
}

func NewConnectionPool(maxConns int, factory func(context.Context) (io.Closer, error)) *ConnectionPool {
    p := &ConnectionPool{
        maxConns:    maxConns,
        factory:     factory,
        idleTimeout: 30 * time.Second,
        idle:        list.New(),
        sem:         make(chan struct{}, maxConns),
    }

    // Fill semaphore
    for i := 0; i < maxConns; i++ {
        p.sem <- struct{}{}
    }

    // Start idle sweeper
    go p.sweepLoop()

    return p
}

func (p *ConnectionPool) Acquire(ctx context.Context) (io.Closer, func(), error) {
    // Wait for permit
    select {
    case <-p.sem:
    case <-ctx.Done():
        return nil, nil, ctx.Err()
    }

    p.mu.Lock()

    // Try to reuse idle connection (LIFO for better locality)
    now := time.Now()
    for e := p.idle.Back(); e != nil; e = p.idle.Back() {
        p.idle.Remove(e)
        pc := e.Value.(*pooledConn)

        if now.Sub(pc.lastUsed) < p.idleTimeout {
            p.mu.Unlock()
            return pc.conn, p.returnFunc(pc.conn), nil
        }

        // Stale, dispose
        pc.conn.Close()
        p.live--
    }

    p.mu.Unlock()

    // Create new connection
    conn, err := p.factory(ctx)
    if err != nil {
        p.sem <- struct{}{} // Return permit
        return nil, nil, err
    }

    p.mu.Lock()
    p.live++
    p.mu.Unlock()

    return conn, p.returnFunc(conn), nil
}

func (p *ConnectionPool) returnFunc(conn io.Closer) func() {
    return func() {
        p.mu.Lock()
        p.idle.PushBack(&pooledConn{conn: conn, lastUsed: time.Now()})
        p.mu.Unlock()
        p.sem <- struct{}{}
    }
}

func (p *ConnectionPool) sweepLoop() {
    ticker := time.NewTicker(p.idleTimeout / 2)
    defer ticker.Stop()

    for range ticker.C {
        p.sweep()
    }
}

func (p *ConnectionPool) sweep() {
    p.mu.Lock()
    defer p.mu.Unlock()

    now := time.Now()
    var next *list.Element
    for e := p.idle.Front(); e != nil; e = next {
        next = e.Next()
        pc := e.Value.(*pooledConn)

        if now.Sub(pc.lastUsed) >= p.idleTimeout {
            p.idle.Remove(e)
            pc.conn.Close()
            p.live--
        }
    }
}
```

---

## 8. Multi-Provider Failover in Go

```go
package provider

import (
    "context"
    "errors"
    "io"
    "sort"
)

type ProviderType int

const (
    Primary ProviderType = iota
    Backup
    Disabled
)

type Provider struct {
    Type                 ProviderType
    Client               NntpClient
    AvailableConnections int
}

type MultiProviderClient struct {
    providers []*Provider
}

func (m *MultiProviderClient) DecodedBody(ctx context.Context, segmentID string) (io.ReadCloser, error) {
    providers := m.orderedProviders()

    var lastErr error
    for i, p := range providers {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        stream, err := p.Client.DecodedBody(ctx, segmentID)
        if err != nil {
            lastErr = err
            continue
        }

        // Check for article not found
        if stream == nil {
            if i < len(providers)-1 {
                continue // Try next provider
            }
            return nil, ErrArticleNotFound
        }

        return stream, nil
    }

    if lastErr != nil {
        return nil, lastErr
    }
    return nil, errors.New("no providers available")
}

func (m *MultiProviderClient) orderedProviders() []*Provider {
    active := make([]*Provider, 0, len(m.providers))
    for _, p := range m.providers {
        if p.Type != Disabled {
            active = append(active, p)
        }
    }

    sort.Slice(active, func(i, j int) bool {
        if active[i].Type != active[j].Type {
            return active[i].Type < active[j].Type
        }
        return active[i].AvailableConnections > active[j].AvailableConnections
    })

    return active
}

var ErrArticleNotFound = errors.New("article not found")
```

---

## 9. AES Decryption for 7z in Go

7z archives use AES-256-CBC with a special key derivation.

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/sha256"
    "encoding/binary"
    "errors"
    "io"
    "unicode/utf16"
)

type AesParams struct {
    DecodedSize int64
    IV          [16]byte
    Key         [32]byte
}

// Derive7zKey derives the AES key from password using 7z's algorithm
func Derive7zKey(password string, salt []byte, numCycles int) [32]byte {
    // Convert password to UTF-16LE
    passRunes := []rune(password)
    passUtf16 := utf16.Encode(passRunes)
    passBytes := make([]byte, len(passUtf16)*2)
    for i, r := range passUtf16 {
        binary.LittleEndian.PutUint16(passBytes[i*2:], r)
    }

    // Special case: direct copy
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

        // Increment 8-byte counter
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

// ParseAesInfo extracts numCycles, salt, and IV from 7z coder info
func ParseAesInfo(info []byte) (numCycles int, salt []byte, iv [16]byte) {
    bt := info[0]
    numCycles = int(bt & 0x3F)

    if (bt & 0xC0) == 0 {
        return numCycles, nil, iv
    }

    bt2 := info[1]
    saltSize := int(((bt >> 7) & 1) + (bt2 >> 4))
    ivSize := int(((bt >> 6) & 1) + (bt2 & 15))

    salt = info[2 : 2+saltSize]
    copy(iv[:], info[2+saltSize:2+saltSize+ivSize])

    return numCycles, salt, iv
}

// AesDecoderStream provides seekable AES-CBC decryption
type AesDecoderStream struct {
    stream      io.ReadSeeker
    key         [32]byte
    baseIV      [16]byte
    decodedSize int64

    cipher cipher.BlockMode
    pos    int64
    buf    []byte
}

const blockSize = 16

func NewAesDecoderStream(stream io.ReadSeeker, params AesParams) *AesDecoderStream {
    block, _ := aes.NewCipher(params.Key[:])

    return &AesDecoderStream{
        stream:      stream,
        key:         params.Key,
        baseIV:      params.IV,
        decodedSize: params.DecodedSize,
        cipher:      cipher.NewCBCDecrypter(block, params.IV[:]),
    }
}

func (s *AesDecoderStream) Seek(offset int64, whence int) (int64, error) {
    var target int64
    switch whence {
    case io.SeekStart:
        target = offset
    case io.SeekCurrent:
        target = s.pos + offset
    case io.SeekEnd:
        target = s.decodedSize + offset
    }

    if target < 0 || target > s.decodedSize {
        return s.pos, errors.New("seek out of bounds")
    }

    blockIndex := target / blockSize
    blockOffset := int(target % blockSize)

    // Determine IV: previous ciphertext block or base IV
    var iv [16]byte
    if blockIndex > 0 {
        s.stream.Seek((blockIndex-1)*blockSize, io.SeekStart)
        io.ReadFull(s.stream, iv[:])
    } else {
        s.stream.Seek(0, io.SeekStart)
        iv = s.baseIV
    }

    // Create new decryptor
    block, _ := aes.NewCipher(s.key[:])
    s.cipher = cipher.NewCBCDecrypter(block, iv[:])
    s.buf = nil
    s.pos = target - int64(blockOffset)

    // If seeking into middle of block, decrypt and buffer remainder
    if blockOffset > 0 {
        cipherBlock := make([]byte, blockSize)
        io.ReadFull(s.stream, cipherBlock)

        plainBlock := make([]byte, blockSize)
        s.cipher.CryptBlocks(plainBlock, cipherBlock)

        s.buf = plainBlock[blockOffset:]
        s.pos = target
    }

    return target, nil
}

func (s *AesDecoderStream) Read(p []byte) (n int, err error) {
    // First return any buffered data
    if len(s.buf) > 0 {
        n = copy(p, s.buf)
        s.buf = s.buf[n:]
        s.pos += int64(n)
        return n, nil
    }

    if s.pos >= s.decodedSize {
        return 0, io.EOF
    }

    // Read and decrypt blocks
    remaining := int(s.decodedSize - s.pos)
    toRead := len(p)
    if toRead > remaining {
        toRead = remaining
    }

    // Round up to block size
    blocksNeeded := (toRead + blockSize - 1) / blockSize
    cipherData := make([]byte, blocksNeeded*blockSize)

    _, err = io.ReadFull(s.stream, cipherData)
    if err != nil {
        return 0, err
    }

    plainData := make([]byte, len(cipherData))
    s.cipher.CryptBlocks(plainData, cipherData)

    n = copy(p, plainData[:toRead])
    s.pos += int64(n)

    // Buffer any remainder
    if len(plainData) > toRead {
        s.buf = plainData[toRead:]
    }

    return n, nil
}
```

---

## 10. File Type Detection in Go

```go
package detect

import "bytes"

var signatures = []struct {
    magic    []byte
    fileType string
}{
    {[]byte("Rar!\x1a\x07\x01\x00"), "rar5"},
    {[]byte("Rar!\x1a\x07\x00"), "rar"},
    {[]byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}, "7z"},
    {[]byte("PK\x03\x04"), "zip"},
    {[]byte{0x1A, 0x45, 0xDF, 0xA3}, "mkv"},
    {[]byte{0x00, 0x00, 0x00, 0x1C, 0x66, 0x74, 0x79, 0x70}, "mp4"},
}

func DetectFileType(header []byte) string {
    for _, sig := range signatures {
        if bytes.HasPrefix(header, sig.magic) {
            return sig.fileType
        }
    }
    return "unknown"
}
```

---

## 11. Key Go Patterns Used

### Context for Cancellation
All long-running operations accept `context.Context` for cancellation:
```go
func (c *Client) Body(ctx context.Context, messageID string) (io.ReadCloser, error)
```

### io.Reader/io.ReadCloser Interfaces
Streams implement standard Go interfaces for composability:
```go
type NzbFileStream struct { ... }
func (s *NzbFileStream) Read(p []byte) (n int, err error)
func (s *NzbFileStream) Seek(offset int64, whence int) (int64, error)
func (s *NzbFileStream) Close() error
```

### Channels for Concurrent Prefetch
Use buffered channels for the segment download queue:
```go
streamChan: make(chan io.ReadCloser, bufferSize),
```

### sync.Mutex for Thread Safety
Protect shared state with mutexes:
```go
s.mu.Lock()
defer s.mu.Unlock()
```

### Error Wrapping
Use `fmt.Errorf` with `%w` for error chains:
```go
return nil, fmt.Errorf("failed to fetch segment %s: %w", id, err)
```

### Goroutines for Background Work
Use goroutines for concurrent operations:
```go
go s.downloadSegments()
```

### defer for Cleanup
Use defer for resource cleanup:
```go
defer close(s.streamChan)
defer ticker.Stop()
```
