package usenet_pool

import (
	"bytes"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

var ftLog = logger.Scoped("nntp/file_type")

type FileType int

const (
	FileTypePlain FileType = iota
	FileTypeRar
	FileType7z
)

func (ft FileType) String() string {
	switch ft {
	case FileTypePlain:
		return "plain"
	case FileTypeRar:
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

func DetectFileType(fileBytes []byte, filename string) FileType {
	if bytes.HasPrefix(fileBytes, magicBytesRAR5) || bytes.HasPrefix(fileBytes, magicBytesRAR4) {
		ftLog.Trace("detected file type", "filename", filename, "type", FileTypeRar, "method", "magic_bytes")
		return FileTypeRar
	}

	if bytes.HasPrefix(fileBytes, magicBytes7Zip) {
		ftLog.Trace("detected file type", "filename", filename, "type", FileType7z, "method", "magic_bytes")
		return FileType7z
	}

	if rarRegex.MatchString(filename) {
		ftLog.Trace("DetectFileType", "filename", filename, "type", FileTypeRar, "method", "extension")
		return FileTypeRar
	}

	if sevenZipRegex.MatchString(filename) {
		ftLog.Trace("DetectFileType", "filename", filename, "type", FileType7z, "method", "extension")
		return FileType7z
	}

	ftLog.Trace("DetectFileType", "filename", filename, "type", FileTypePlain, "method", "default")
	return FileTypePlain
}

var isVideoFile = func() func(filename string) bool {
	videoExtensions := map[string]struct{}{
		".mkv":  struct{}{},
		".mp4":  struct{}{},
		".avi":  struct{}{},
		".webm": struct{}{},
		".mov":  struct{}{},
		".wmv":  struct{}{},
		".flv":  struct{}{},
		".ts":   struct{}{},
		".m2ts": struct{}{},
		".mpg":  struct{}{},
		".mpeg": struct{}{},
		".m4v":  struct{}{},
	}

	return func(filename string) bool {
		_, found := videoExtensions[strings.ToLower(filepath.Ext(filename))]
		return found
	}
}()

// .part01.rar format
var rarPartNumberRegex = regexp.MustCompile(`(?i)\.part(\d+)\.rar$`)

// .r00, .r01 format (.rar is first part, .r00 is second, etc.)
var rarRNumberRegex = regexp.MustCompile(`(?i)\.r(\d+)$`)

// .rar
var rarFirstPartRegex = regexp.MustCompile(`(?i)\.rar$`)

func GetRARVolumeNumber(filename string) int {
	if matches := rarPartNumberRegex.FindStringSubmatch(filename); len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}

	if matches := rarRNumberRegex.FindStringSubmatch(filename); len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n + 1
	}

	if rarFirstPartRegex.MatchString(filename) {
		return 0
	}

	return -1
}

// Get7zPartNumber extracts the part number from a 7z filename
// Returns -1 if not a recognized 7z part pattern
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

// GetContentType returns the MIME type for a filename
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

// GetBaseArchiveName extracts the base archive name without part indicators
func GetBaseArchiveName(filename string) string {
	lower := strings.ToLower(filename)
	base := path.Base(filename)

	// Remove .part01.rar style suffix
	re := regexp.MustCompile(`(?i)\.part\d+\.rar$`)
	base = re.ReplaceAllString(base, ".rar")

	// Remove .r00, .r01 style suffix -> base.rar
	re = regexp.MustCompile(`(?i)\.r\d+$`)
	if re.MatchString(lower) {
		base = re.ReplaceAllString(base, ".rar")
	}

	// Remove .7z.001 style suffix
	re = regexp.MustCompile(`(?i)\.7z\.\d+$`)
	base = re.ReplaceAllString(base, ".7z")

	return base
}
