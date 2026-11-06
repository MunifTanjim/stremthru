//go:build integration

package qbittorrent

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

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

// FreePort asks the OS for an available TCP port.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// WaitForHTTP polls a URL until it responds or times out.
func WaitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", url)
}

// GetQbitPassword extracts the temporary password from qBit container logs.
func GetQbitPassword(containerName string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "temporary password") {
					parts := strings.Split(line, ": ")
					if len(parts) >= 2 {
						return strings.TrimSpace(parts[len(parts)-1]), nil
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return "", fmt.Errorf("timed out waiting for qBit password in logs")
}
