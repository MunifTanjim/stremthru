package usenet_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileTypeDetection(t *testing.T) {
	t.Run("MagicBytes", func(t *testing.T) {
		t.Run("RAR4", func(t *testing.T) {
			data := append(append([]byte{}, magicBytesRAR4...), make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.rar")
			assert.Equal(t, FileTypeRAR, ft)
		})

		t.Run("RAR5", func(t *testing.T) {
			data := append(append([]byte{}, magicBytesRAR5...), make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.rar")
			assert.Equal(t, FileTypeRAR, ft)
		})

		t.Run("7z", func(t *testing.T) {
			data := append(append([]byte{}, magicBytes7Zip...), make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.7z")
			assert.Equal(t, FileType7z, ft)
		})
	})

	t.Run("ExtensionBased", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected FileType
		}{
			{"movie.mkv", FileTypePlain},
			{"movie.mp4", FileTypePlain},
			{"movie.avi", FileTypePlain},
			{"movie.mov", FileTypePlain},
			{"movie.webm", FileTypePlain},
			{"archive.rar", FileTypeRAR},
			{"archive.r00", FileTypeRAR},
			{"archive.r01", FileTypeRAR},
			{"archive.r99", FileTypeRAR},
			{"archive.part01.rar", FileTypeRAR},
			{"archive.part99.rar", FileTypeRAR},
			{"archive.7z", FileType7z},
			{"archive.7z.001", FileType7z},
			{"archive.7z.002", FileType7z},
			{"unknown.txt", FileTypePlain},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				// Use non-magic bytes
				data := []byte("some random data")
				ft := DetectFileType(data, tc.filename)
				assert.Equal(t, tc.expected, ft, "filename: %s", tc.filename)
			})
		}
	})

	t.Run("IsVideoFile", func(t *testing.T) {
		videoFiles := []string{
			"movie.mkv", "MOVIE.MKV", "movie.mp4", "movie.avi",
			"movie.mov", "movie.wmv", "movie.flv", "movie.webm",
			"movie.m4v", "movie.mpg", "movie.mpeg", "movie.ts",
		}
		for _, f := range videoFiles {
			assert.True(t, isVideoFile(f), "should be video: %s", f)
		}

		nonVideoFiles := []string{
			"file.rar", "file.7z", "file.txt", "file.exe", "file.zip",
		}
		for _, f := range nonVideoFiles {
			assert.False(t, isVideoFile(f), "should not be video: %s", f)
		}
	})

	t.Run("GetRarPartNumber", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected int
		}{
			{"archive.rar", 0},
			{"archive.r00", 1},
			{"archive.r01", 2},
			{"archive.r99", 100},
			{"archive.part01.rar", 1},
			{"archive.part02.rar", 2},
			{"archive.part99.rar", 99},
			{"notrar.txt", -1},
			{"archive.zip", -1},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := GetRARVolumeNumber(tc.filename)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Get7zPartNumber", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected int
		}{
			{"archive.7z", 0},
			{"archive.7z.001", 1},
			{"archive.7z.002", 2},
			{"archive.7z.099", 99},
			{"not7z.txt", -1},
			{"archive.zip", -1},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := Get7zVolumeNumber(tc.filename)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("GetContentType", func(t *testing.T) {
		testCases := []struct {
			filename    string
			contentType string
		}{
			{"movie.mkv", "video/x-matroska"},
			{"movie.mp4", "video/mp4"},
			{"movie.avi", "video/x-msvideo"},
			{"movie.webm", "video/webm"},
			{"movie.mov", "video/quicktime"},
			{"movie.wmv", "video/x-ms-wmv"},
			{"movie.flv", "video/x-flv"},
			{"movie.ts", "video/mp2t"},
			{"movie.m2ts", "video/mp2t"},
			{"movie.mpg", "video/mpeg"},
			{"movie.mpeg", "video/mpeg"},
			{"movie.m4v", "video/x-m4v"},
			{"unknown.xyz", "application/octet-stream"},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := GetContentType(tc.filename)
				assert.Equal(t, tc.contentType, result)
			})
		}
	})
}
