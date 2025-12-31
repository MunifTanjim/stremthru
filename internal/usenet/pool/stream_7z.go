package usenet_pool

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/bodgit/sevenzip"
	"golang.org/x/crypto/pbkdf2"
)

var sevenZLog = logger.Scoped("nntp/7z")

type AesParams struct {
	DecodedSize int64    // Size after decryption
	IV          [16]byte // Initialization vector
	Key         [32]byte // Derived AES-256 key
}

// SevenZFileEntry represents a file inside a 7z archive
type SevenZFileEntry struct {
	Name         string
	UnpackedSize int64
	IsDir        bool
	Stream       int // Stream index (files with same stream are in same compressed block)
}

// SevenZArchiveInfo contains parsed information about a 7z archive
type SevenZArchiveInfo struct {
	Entries []SevenZFileEntry
	Solid   bool // True if archive uses solid compression (multiple files share streams)
}

// SevenZPartInfo contains information about a single 7z archive part
type SevenZPartInfo struct {
	PartNumber int
	PartSize   int64
	Segments   []nzb.Segment // Segment message IDs
	Groups     []string      // Newsgroups
}

// SevenZMultipartInfo contains information about a multi-part 7z archive
type SevenZMultipartInfo struct {
	Parts      []SevenZPartInfo
	Files      []SevenZExtractedFile
	Streamable bool // True if files can be streamed directly
}

// SevenZExtractedFile represents a file that can be extracted/streamed from the archive
type SevenZExtractedFile struct {
	Name         string
	Size         int64
	Parts        []SevenZFilePart
	IsStreamable bool // True if can be streamed without decompression
}

// SevenZFilePart represents a portion of a file within an archive part
type SevenZFilePart struct {
	PartIndex       int       // Index in SevenZMultipartInfo.Parts
	ByteRangeInPart ByteRange // Where the data is within this 7z part
	ByteRangeInFile ByteRange // What portion of the final file this covers
}

// Parse7zHeaders reads 7z headers from a ReaderAt to extract file entries
func Parse7zHeaders(r io.ReaderAt, size int64, password string) (*SevenZArchiveInfo, error) {
	var reader *sevenzip.Reader
	var err error

	sevenZLog.Trace("Parse7zHeaders opening archive", "size", size)

	if password != "" {
		reader, err = sevenzip.NewReaderWithPassword(r, size, password)
	} else {
		reader, err = sevenzip.NewReader(r, size)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open 7z: %w", err)
	}

	info := &SevenZArchiveInfo{
		Entries: []SevenZFileEntry{},
	}

	// Track streams to detect solid compression
	streamCounts := make(map[int]int)

	for _, file := range reader.File {
		entry := SevenZFileEntry{
			Name:         file.Name,
			UnpackedSize: int64(file.UncompressedSize),
			IsDir:        file.FileInfo().IsDir(),
			Stream:       file.Stream,
		}

		if !entry.IsDir {
			streamCounts[file.Stream]++
		}

		info.Entries = append(info.Entries, entry)
	}

	// If any stream has more than one file, it's solid compression
	for _, count := range streamCounts {
		if count > 1 {
			info.Solid = true
			break
		}
	}

	sevenZLog.Trace("Parse7zHeaders complete", "file_count", len(info.Entries))

	return info, nil
}

// Group7zParts groups 7z files by their base archive name and orders by part number
func Group7zParts(files []SevenZPartInfo) []SevenZPartInfo {
	// Sort by part number
	sorted := make([]SevenZPartInfo, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PartNumber < sorted[j].PartNumber
	})
	return sorted
}

// Is7zStreamable checks if a 7z archive can be streamed directly
// 7z archives are generally not directly streamable because the library
// doesn't expose raw stream offsets. Returns false for all 7z archives.
// Use the sevenzip library's Open() method to extract files instead.
func Is7zStreamable(info *SevenZArchiveInfo) bool {
	// 7z format doesn't support direct streaming like RAR store mode
	// Files must be extracted through the sevenzip library
	return false
}

// FindVideoFilesIn7z returns entries for video files in the archive
func FindVideoFilesIn7z(info *SevenZArchiveInfo) []SevenZFileEntry {
	var videos []SevenZFileEntry
	for _, entry := range info.Entries {
		if entry.IsDir {
			continue
		}
		if isVideoFile(entry.Name) {
			videos = append(videos, entry)
		}
	}
	return videos
}

// Derive7zKey derives an AES-256 key from password using PBKDF2
// This matches the key derivation used by 7z AES encryption
func Derive7zKey(password string, salt []byte, iterations int) [32]byte {
	// 7z uses PBKDF2-HMAC-SHA256 for key derivation
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
	var result [32]byte
	copy(result[:], key)
	return result
}

// NewAesDecryptingReader creates a reader that decrypts AES-256-CBC encrypted data
func NewAesDecryptingReader(r io.Reader, params *AesParams) (io.Reader, error) {
	block, err := aes.NewCipher(params.Key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	return &aesDecryptingReader{
		reader:      r,
		blockMode:   cipher.NewCBCDecrypter(block, params.IV[:]),
		decodedSize: params.DecodedSize,
		buffer:      make([]byte, 0, aes.BlockSize*16),
	}, nil
}

// aesDecryptingReader decrypts AES-CBC encrypted data
type aesDecryptingReader struct {
	reader      io.Reader
	blockMode   cipher.BlockMode
	decodedSize int64
	bytesRead   int64
	buffer      []byte
	bufferPos   int
	eof         bool
}

func (r *aesDecryptingReader) Read(p []byte) (n int, err error) {
	if r.bytesRead >= r.decodedSize {
		return 0, io.EOF
	}

	// If we have buffered data, return from that first
	if r.bufferPos < len(r.buffer) {
		n = copy(p, r.buffer[r.bufferPos:])
		r.bufferPos += n
		r.bytesRead += int64(n)

		// Check if we've reached the decoded size
		if r.bytesRead >= r.decodedSize {
			return n, io.EOF
		}
		return n, nil
	}

	// Read encrypted blocks
	blockSize := r.blockMode.BlockSize()
	encryptedBuf := make([]byte, blockSize*16) // Read multiple blocks at once

	bytesRead, err := io.ReadFull(r.reader, encryptedBuf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, err
	}

	if bytesRead == 0 {
		r.eof = true
		return 0, io.EOF
	}

	// Ensure we have complete blocks
	if bytesRead%blockSize != 0 {
		// Pad to complete block
		padding := blockSize - (bytesRead % blockSize)
		bytesRead += padding
	}

	// Decrypt
	decrypted := encryptedBuf[:bytesRead]
	r.blockMode.CryptBlocks(decrypted, decrypted)

	// Store in buffer and remove PKCS7 padding if this is the last block
	r.buffer = decrypted
	r.bufferPos = 0

	// Copy to output
	remaining := r.decodedSize - r.bytesRead
	toCopy := int64(len(p))
	if toCopy > remaining {
		toCopy = remaining
	}
	if toCopy > int64(len(r.buffer)-r.bufferPos) {
		toCopy = int64(len(r.buffer) - r.bufferPos)
	}

	n = copy(p, r.buffer[r.bufferPos:r.bufferPos+int(toCopy)])
	r.bufferPos += n
	r.bytesRead += int64(n)

	if r.bytesRead >= r.decodedSize {
		return n, io.EOF
	}
	return n, nil
}
