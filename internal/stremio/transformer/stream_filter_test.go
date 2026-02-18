package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/stretchr/testify/assert"
)

func TestStreamFilter_Match_Resolution(t *testing.T) {
	for _, tc := range []struct {
		name     string
		filter   StreamFilterBlob
		result   *StreamExtractorResult
		expected bool
	}{
		{
			name:   "equal",
			filter: `Resolution == "1080p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: true,
		},
		{
			name:   "not equal",
			filter: `Resolution != "1080p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: false,
		},
		{
			name:   "greater than",
			filter: `Resolution > "720p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: true,
		},
		{
			name:   "greater than or equal",
			filter: `Resolution >= "1080p" && Resolution >= "720p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: true,
		},
		{
			name:   "less than",
			filter: `Resolution < "720p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: false,
		},
		{
			name:   "less than or equal",
			filter: `Resolution <= "1080p" && Resolution <= "720p"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sf, err := tc.filter.Parse()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, sf.Match(tc.result))
		})
	}
}

func TestStreamFilter_Match_Quality(t *testing.T) {
	for _, tc := range []struct {
		name     string
		filter   StreamFilterBlob
		result   *StreamExtractorResult
		expected bool
	}{
		{
			name:   "equal",
			filter: `Quality == "BluRay"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "BluRay",
				},
			},
			expected: true,
		},
		{
			name:   "not equal",
			filter: `Quality != "WEB-DL"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "WEB-DL",
				},
			},
			expected: false,
		},
		{
			name:   "greater than",
			filter: `Quality > "HDTV"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "BluRay",
				},
			},
			expected: true,
		},
		{
			name:   "greater than or equal",
			filter: `Quality >= "WEB-DL" && Quality >= "HDTV"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "WEB-DL",
				},
			},
			expected: true,
		},
		{
			name:   "less than",
			filter: `Quality < "BluRay"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "WEB-DL",
				},
			},
			expected: true,
		},
		{
			name:   "less than or equal",
			filter: `Quality <= "WEB-DL" && Quality <= "HDTV"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Quality: "WEB-DL",
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sf, err := tc.filter.Parse()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, sf.Match(tc.result))
		})
	}
}

func TestStreamFilter_Match_Size(t *testing.T) {
	for _, tc := range []struct {
		name     string
		filter   StreamFilterBlob
		result   *StreamExtractorResult
		expected bool
	}{
		{
			name:   "equal",
			filter: `Size == "1.5 GB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Size: "1.5 GB",
				},
			},
			expected: true,
		},
		{
			name:   "not equal",
			filter: `Size != "700 MB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Size: "700 MB",
				},
			},
			expected: false,
		},
		{
			name:   "greater than",
			filter: `Size > "700 MB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Size: "1.5 GB",
				},
			},
			expected: true,
		},
		{
			name:   "less than",
			filter: `Size < "700 MB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{
					Size: "1.5 GB",
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sf, err := tc.filter.Parse()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, sf.Match(tc.result))
		})
	}
}

func TestStreamFilter_Match_FileSize(t *testing.T) {
	for _, tc := range []struct {
		name     string
		filter   StreamFilterBlob
		result   *StreamExtractorResult
		expected bool
	}{
		{
			name:   "greater than",
			filter: `File.Size > "700 MB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{},
				File: StreamExtractorResultFile{
					Size: "1.5 GB",
				},
			},
			expected: true,
		},
		{
			name:   "less than",
			filter: `File.Size < "700 MB"`,
			result: &StreamExtractorResult{
				Result: &ptt.Result{},
				File: StreamExtractorResultFile{
					Size: "1.5 GB",
				},
			},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sf, err := tc.filter.Parse()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, sf.Match(tc.result))
		})
	}
}

func TestStreamFilter_QualityRank(t *testing.T) {
	for _, tc := range []struct {
		input string
		rank  int64
	}{
		{"BluRay-REMUX", 100},
		{"Remux", 100},
		{"BluRay-Mux", 99},
		{"MUX", 99},
		{"BluRay", 98},
		{"BRRip", 96},
		{"BDRip", 94},
		{"BD", 94},
		{"UHDRip", 92},
		{"UHD", 92},
		{"WEB-DL", 89},
		{"WEBDL", 89},
		{"WEB-Rip", 85},
		{"WEBRip", 85},
		{"WEB", 80},
		{"HDRip", 79},
		{"HDTV", 75},
		{"DVDRip", 60},
		{"DVD", 60},
		{"SATRip", 59},
		{"PDTV", 57},
		{"PPV", 55},
		{"CAM", 40},
		{"TeleSync", 30},
		{"SCR", 20},
		{"Unknown", 0},
	} {
		assert.Equal(t, getQualityRank(tc.input), tc.rank)
	}
}
