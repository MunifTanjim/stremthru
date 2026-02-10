//go:build integration

package qbittorrent

// Test helpers that expose internal types for integration tests in other packages.
// Only compiled with the integration build tag.

// TestConfig wraps qbitConfig for use in external integration tests.
type TestConfig = qbitConfig

// NewAPIClientForTest creates an APIClient usable from external test packages.
func NewAPIClientForTest() *APIClient {
	return NewAPIClient(&APIClientConfig{})
}

// NewConfigForTest creates a qbitConfig usable from external test packages.
func NewConfigForTest(url, username, password, fileBaseURL string) *TestConfig {
	return &qbitConfig{
		URL:         url,
		Username:    username,
		Password:    password,
		FileBaseURL: fileBaseURL,
	}
}
