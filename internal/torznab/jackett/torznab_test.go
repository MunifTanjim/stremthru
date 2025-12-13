package jackett

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTorznabURL_FromEncoded(t *testing.T) {
	for _, test := range []struct {
		input           string
		expectedIndexer string
		expectedBase    string
		shouldError     bool
		errorMsg        string
	}{
		// old format
		{
			input:           "indexer123:::http://localhost:9117",
			expectedIndexer: "indexer123",
			expectedBase:    "http://localhost:9117",
			shouldError:     false,
		},
		{
			input:           "all:::https://jackett.example.com",
			expectedIndexer: "all",
			expectedBase:    "https://jackett.example.com",
			shouldError:     false,
		},
		// new format
		{
			input:           "http:localhost:9117::all",
			expectedIndexer: "all",
			expectedBase:    "http://localhost:9117",
			shouldError:     false,
		},
		{
			input:           "https:example.com::myindexer",
			expectedIndexer: "myindexer",
			expectedBase:    "https://example.com",
			shouldError:     false,
		},
		{
			input:           "http:localhost:9117/path::indexer",
			expectedIndexer: "indexer",
			expectedBase:    "http://localhost:9117/path",
			shouldError:     false,
		},
		// error
		{
			input:       "invalid-no-separator",
			shouldError: true,
			errorMsg:    "invalid encoded torznab url",
		},
		{
			input:       "",
			shouldError: true,
			errorMsg:    "invalid encoded torznab url",
		},
	} {
		turl := TorznabURL(test.input)
		err := turl.FromEncoded()

		if test.shouldError {
			assert.Error(t, err, "input: %s", test.input)
			assert.Equal(t, test.errorMsg, err.Error(), "input: %s", test.input)
		} else {
			assert.NoError(t, err, "input: %s", test.input)
			assert.Equal(t, test.expectedIndexer, turl.IndexerId, "input: %s", test.input)
			assert.Equal(t, test.expectedBase, turl.BaseURL, "input: %s", test.input)
		}
	}
}

func TestTorznabURL_Encode(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected string
	}{
		// old encoded format
		{
			input:    "indexer123:::http://localhost:9117",
			expected: "http:localhost:9117::indexer123",
		},
		// new encoded format
		{
			input:    "http:localhost:9117::all",
			expected: "http:localhost:9117::all",
		},
		// url
		{
			input:    "http://localhost:9117/api/v2.0/indexers/all/results/torznab",
			expected: "http:localhost:9117::all",
		},
		{
			input:    "https://jackett.example.com/api/v2.0/indexers/myindexer/results/torznab/",
			expected: "https:jackett.example.com::myindexer",
		},
		{
			input:    "http://invalid-url",
			expected: "",
		},
	} {
		turl := TorznabURL(test.input)
		result := turl.Encode()
		assert.Equal(t, test.expected, result, "input: %s", test.input)
	}
}
