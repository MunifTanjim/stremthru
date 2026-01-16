package usenet_pool

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

var ftLog = logger.Scoped("usenet/pool/file_type")

type FileType int

const (
	FileTypePlain FileType = iota
	FileTypeRAR
	FileType7z
)

func (ft FileType) String() string {
	switch ft {
	case FileTypePlain:
		return "plain"
	case FileTypeRAR:
		return "rar"
	case FileType7z:
		return "7z"
	default:
		return "unknown"
	}
}

var (
	magicBytesRAR4 = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}
	magicBytesRAR5 = []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}
	magicBytes7Zip = []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}
)

// RAR patterns: .rar, .r00, .r01, .part01.rar
var rarRegex = regexp.MustCompile(`(?i)\.r(ar|\d+)$`)

// 7z patterns: .7z, .7z.001, .7z.002
var sevenZipRegex = regexp.MustCompile(`(?i)\.7z(\.\d+)?$`)

func DetectArchiveFileTypeByExtension(filename string) FileType {
	if rarRegex.MatchString(filename) {
		return FileTypeRAR
	}
	if sevenZipRegex.MatchString(filename) {
		return FileType7z
	}
	return FileTypePlain
}

func DetectFileType(fileBytes []byte, filename string) FileType {
	if bytes.HasPrefix(fileBytes, magicBytesRAR5) || bytes.HasPrefix(fileBytes, magicBytesRAR4) {
		ftLog.Trace("file type - detected", "filename", filename, "type", FileTypeRAR, "method", "magic_bytes")
		return FileTypeRAR
	}

	if bytes.HasPrefix(fileBytes, magicBytes7Zip) {
		ftLog.Trace("file type - detected", "filename", filename, "type", FileType7z, "method", "magic_bytes")
		return FileType7z
	}

	ft := DetectArchiveFileTypeByExtension(filename)
	ftLog.Trace("file type - detected", "filename", filename, "type", FileTypePlain, "method", "default")
	return ft
}

var isVideoFile = func() func(filename string) bool {
	videoExtensions := map[string]struct{}{
		".mkv":  {},
		".mp4":  {},
		".avi":  {},
		".webm": {},
		".mov":  {},
		".wmv":  {},
		".flv":  {},
		".ts":   {},
		".m2ts": {},
		".mpg":  {},
		".mpeg": {},
		".m4v":  {},
	}

	return func(filename string) bool {
		_, found := videoExtensions[strings.ToLower(filepath.Ext(filename))]
		return found
	}
}()

func GetContentType(filename string) string {
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
	case strings.HasSuffix(lower, ".mpg"), strings.HasSuffix(lower, ".mpeg"):
		return "video/mpeg"
	case strings.HasSuffix(lower, ".m4v"):
		return "video/x-m4v"
	default:
		return "application/octet-stream"
	}
}

func IsArchiveFile(filename string) bool {
	switch ft := DetectArchiveFileTypeByExtension(filename); ft {
	case FileType7z, FileTypeRAR:
		return true
	default:
		return false
	}
}
