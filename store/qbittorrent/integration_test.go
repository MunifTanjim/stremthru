//go:build integration

package qbittorrent

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/store"
	"github.com/stretchr/testify/suite"
)

// Integration tests for the qBittorrent store.
//
// By default, the test suite auto-manages a qBittorrent Docker container.
// Just run:
//
//	go test -tags integration ./store/qbittorrent/ -v
//
// To use an existing qBittorrent instance instead, set environment variables:
//
//	QBIT_URL       - qBittorrent WebUI URL (e.g. http://localhost:18080)
//	QBIT_USER      - WebUI username (default: admin)
//	QBIT_PASS      - WebUI password (required when using external instance)
//	QBIT_FILE_URL  - File server URL (default: same as QBIT_URL)

// Test magnet: Ubuntu 24.04.3 LTS ISO (public, well-seeded, legal)
const testMagnet = "magnet:?xt=urn:btih:d160b8d8ea35a5b4e52837468fc8f03d55cef1f7&dn=ubuntu-24.04.3-desktop-amd64.iso"
const testHash = "d160b8d8ea35a5b4e52837468fc8f03d55cef1f7"

const qbitContainerName = "qbit-integ-test"

// freePort asks the OS for an available port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

type QBitIntegrationSuite struct {
	suite.Suite
	client      *APIClient
	cfg         *qbitConfig
	managedQbit bool // true if we started the qBit container
}

// waitForFiles polls until qBit has resolved the torrent metadata and files are available.
func (s *QBitIntegrationSuite) waitForFiles(hash string) []TorrentFile {
	for i := 0; i < 30; i++ {
		files, err := s.client.GetTorrentFiles(s.cfg, hash)
		if err == nil && len(files) > 0 {
			return files
		}
		time.Sleep(1 * time.Second)
	}
	s.Require().Fail("timed out waiting for torrent files to become available")
	return nil
}

// startContainer starts a Docker container and returns a cleanup function.
func startContainer(name string, args ...string) (string, error) {
	// Remove any leftover container with the same name
	_ = exec.Command("docker", "rm", "-f", name).Run()

	cmdArgs := append([]string{"run", "-d", "--name", name}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %s: %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// waitForHTTP polls a URL until it responds or times out.
func waitForHTTP(url string, timeout time.Duration) error {
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

// getQbitPassword extracts the temporary password from qBit container logs.
func getQbitPassword(containerName string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "temporary password") {
					// Format: "...A temporary password is provided for this session: XXXXX"
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

func (s *QBitIntegrationSuite) SetupSuite() {
	s.client = NewAPIClient(&APIClientConfig{})

	if os.Getenv("QBIT_URL") != "" {
		// User provided an external qBit instance
		s.cfg = &qbitConfig{
			URL:         strings.TrimRight(os.Getenv("QBIT_URL"), "/"),
			Username:    os.Getenv("QBIT_USER"),
			Password:    os.Getenv("QBIT_PASS"),
			FileBaseURL: strings.TrimRight(os.Getenv("QBIT_FILE_URL"), "/"),
		}
		if s.cfg.Username == "" {
			s.cfg.Username = "admin"
		}
		if s.cfg.FileBaseURL == "" {
			s.cfg.FileBaseURL = s.cfg.URL
		}
		s.Require().NotEmpty(s.cfg.Password, "QBIT_PASS is required when using an external instance")
		return
	}

	// Auto-manage a qBit container
	s.managedQbit = true

	port, err := freePort()
	s.Require().NoError(err, "could not find a free port")
	s.T().Logf("Starting qBittorrent container on port %d...", port)

	_, err = startContainer(qbitContainerName,
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("WEBUI_PORT=%d", port),
		"-e", "PUID=1000",
		"-e", "PGID=1000",
		"lscr.io/linuxserver/qbittorrent:latest",
	)
	s.Require().NoError(err, "failed to start qBit container")

	// Wait for the WebUI to be ready and extract password
	qbitURL := fmt.Sprintf("http://localhost:%d", port)
	s.Require().NoError(waitForHTTP(qbitURL, 30*time.Second), "qBit WebUI did not become ready")

	pass, err := getQbitPassword(qbitContainerName, 30*time.Second)
	s.Require().NoError(err, "could not get qBit password from logs")

	s.cfg = &qbitConfig{
		URL:         qbitURL,
		Username:    "admin",
		Password:    pass,
		FileBaseURL: qbitURL,
	}
	s.T().Logf("qBit container ready (password: %s)", pass)
}

func (s *QBitIntegrationSuite) TearDownSuite() {
	// Clean up test torrent
	_ = s.client.DeleteTorrents(s.cfg, []string{testHash}, true)

	// Stop managed container
	if s.managedQbit {
		s.T().Log("Stopping qBittorrent container...")
		_ = exec.Command("docker", "rm", "-f", qbitContainerName).Run()
	}
}

// startNginx starts an nginx container serving the given directory on a random free port.
// Returns the nginx base URL and a cleanup function.
func (s *QBitIntegrationSuite) startNginx(name string, hostDir string) (nginxURL string, cleanup func()) {
	port, err := freePort()
	s.Require().NoError(err, "could not find a free port for nginx")

	_, err = startContainer(name,
		"--rm",
		"-p", fmt.Sprintf("%d:80", port),
		"-v", hostDir+":/usr/share/nginx/html:ro",
		"nginx:alpine",
	)
	s.Require().NoError(err, "failed to start nginx")

	cleanup = func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	}

	nginxURL = fmt.Sprintf("http://localhost:%d", port)
	err = waitForHTTP(nginxURL, 10*time.Second)
	s.Require().NoError(err, "nginx did not become ready")

	return nginxURL, cleanup
}

// --- Tests ---

func (s *QBitIntegrationSuite) TestA_GetVersion() {
	version, err := s.client.GetVersion(s.cfg)
	s.Require().NoError(err)
	s.NotEmpty(version)
	s.True(strings.HasPrefix(version, "v"), "version should start with 'v', got: %s", version)
}

func (s *QBitIntegrationSuite) TestB_LoginCreatesSession() {
	s.client.invalidateSession(s.cfg)

	client, err := s.client.login(s.cfg)
	s.Require().NoError(err)
	s.NotNil(client)

	key := s.client.sessionKey(s.cfg)
	entry, ok := s.client.sessions.Load(key)
	s.True(ok)
	se := entry.(*sessionEntry)
	s.True(se.expires.After(time.Now()))
}

func (s *QBitIntegrationSuite) TestC_LoginBadCredentials() {
	badCfg := &qbitConfig{
		URL:      s.cfg.URL,
		Username: "wronguser",
		Password: "wrongpass",
	}
	_, err := s.client.login(badCfg)
	s.Error(err)
}

func (s *QBitIntegrationSuite) TestD_AddTorrent() {
	_ = s.client.DeleteTorrents(s.cfg, []string{testHash}, true)
	time.Sleep(1 * time.Second)

	err := s.client.AddTorrentMagnet(s.cfg, testMagnet)
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)

	torrents, err := s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	s.Require().Len(torrents, 1)
	s.Equal(strings.ToLower(testHash), strings.ToLower(torrents[0].Hash))
	s.True(torrents[0].SeqDl, "sequential download should be enabled")
}

func (s *QBitIntegrationSuite) TestE_GetTorrents() {
	torrents, err := s.client.GetTorrents(s.cfg, nil, 10, 0)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(torrents), 1)

	torrents, err = s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	s.Len(torrents, 1)

	torrents, err = s.client.GetTorrents(s.cfg, []string{"0000000000000000000000000000000000000000"}, 0, 0)
	s.Require().NoError(err)
	s.Len(torrents, 0)
}

func (s *QBitIntegrationSuite) TestF_GetTorrentFiles() {
	files := s.waitForFiles(testHash)
	s.GreaterOrEqual(len(files), 1)
	s.NotEmpty(files[0].Name)
	s.Greater(files[0].Size, int64(0))
}

func (s *QBitIntegrationSuite) TestG_SessionReuse() {
	_, err := s.client.GetVersion(s.cfg)
	s.Require().NoError(err)

	key := s.client.sessionKey(s.cfg)
	entry1, _ := s.client.sessions.Load(key)
	se1 := entry1.(*sessionEntry)

	_, err = s.client.GetVersion(s.cfg)
	s.Require().NoError(err)

	entry2, _ := s.client.sessions.Load(key)
	se2 := entry2.(*sessionEntry)

	s.Equal(se1.client, se2.client, "should reuse the same HTTP client")
}

func (s *QBitIntegrationSuite) TestH_GenerateLink() {
	torrents, err := s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	if len(torrents) == 0 {
		err := s.client.AddTorrentMagnet(s.cfg, testMagnet)
		s.Require().NoError(err)
	}

	files := s.waitForFiles(testHash)

	sc := NewStoreClient(&StoreClientConfig{})
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + s.cfg.FileBaseURL

	// No API key should fail
	lockedLink := LockedFileLink("").create(testHash, files[0].Index)
	_, err = sc.GenerateLink(&store.GenerateLinkParams{Link: lockedLink})
	s.Error(err, "GenerateLink without API key should fail")

	// Valid call should return a well-formed URL
	params := &store.GenerateLinkParams{Link: lockedLink}
	params.APIKey = token
	data, err := sc.GenerateLink(params)
	s.Require().NoError(err)
	s.NotEmpty(data.Link)

	parsedURL, err := url.Parse(data.Link)
	s.Require().NoError(err, "generated link should be a valid URL")
	s.NotEmpty(parsedURL.Scheme)
	s.NotEmpty(parsedURL.Host)

	s.True(strings.HasPrefix(data.Link, s.cfg.FileBaseURL+"/"),
		"link should start with FileBaseURL, got: %s", data.Link)

	// URL path should match the qBit file name
	relPath := strings.TrimPrefix(data.Link, s.cfg.FileBaseURL)
	decodedPath, err := url.PathUnescape(relPath)
	s.Require().NoError(err)
	s.Equal("/"+files[0].Name, decodedPath)

	s.Equal(files[0].GetName(), filepath.Base(decodedPath))

	// Generate links for ALL file indices
	for i, f := range files {
		link := LockedFileLink("").create(testHash, f.Index)
		p := &store.GenerateLinkParams{Link: link}
		p.APIKey = token
		d, err := sc.GenerateLink(p)
		s.Require().NoError(err, "file index %d", f.Index)

		decoded, err := url.PathUnescape(strings.TrimPrefix(d.Link, s.cfg.FileBaseURL))
		s.Require().NoError(err)
		s.Equal("/"+f.Name, decoded, "file index %d", i)
	}
}

func (s *QBitIntegrationSuite) TestH1_GenerateLink_FileServerRoundTrip() {
	files := s.waitForFiles(testHash)

	var requestedPath string
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "fake file content")
	}))
	defer fakeServer.Close()

	sc := NewStoreClient(&StoreClientConfig{})
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + fakeServer.URL

	lockedLink := LockedFileLink("").create(testHash, files[0].Index)
	p := &store.GenerateLinkParams{Link: lockedLink}
	p.APIKey = token
	data, err := sc.GenerateLink(p)
	s.Require().NoError(err)

	resp, err := http.Get(data.Link)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	decodedPath, err := url.PathUnescape(requestedPath)
	s.Require().NoError(err)
	s.Equal("/"+files[0].Name, decodedPath)
}

func (s *QBitIntegrationSuite) TestH15_GenerateLink_PathMapping() {
	files := s.waitForFiles(testHash)

	torrents, err := s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	s.Require().Len(torrents, 1)
	savePath := strings.TrimRight(torrents[0].SavePath, "/")

	var requestedPath string
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeServer.Close()

	externalPath := "/media/torrents"
	pathMap := savePath + ":" + externalPath
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + fakeServer.URL + "|" + pathMap

	sc := NewStoreClient(&StoreClientConfig{})
	lockedLink := LockedFileLink("").create(testHash, files[0].Index)
	p := &store.GenerateLinkParams{Link: lockedLink}
	p.APIKey = token
	data, err := sc.GenerateLink(p)
	s.Require().NoError(err)

	resp, err := http.Get(data.Link)
	s.Require().NoError(err)
	defer resp.Body.Close()

	decodedPath, err := url.PathUnescape(requestedPath)
	s.Require().NoError(err)
	s.Equal(externalPath+"/"+files[0].Name, decodedPath)
}

func (s *QBitIntegrationSuite) TestH16_GenerateLink_NginxRoundTrip() {
	files := s.waitForFiles(testHash)

	tmpDir, err := os.MkdirTemp("", "qbit-nginx-test-")
	s.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	testContent := "nginx-test-content-" + files[0].Name
	for _, f := range files {
		filePath := filepath.Join(tmpDir, f.Name)
		s.Require().NoError(os.MkdirAll(filepath.Dir(filePath), 0755))
		s.Require().NoError(os.WriteFile(filePath, []byte("nginx-test-content-"+f.Name), 0644))
	}

	nginxURL, cleanup := s.startNginx("nginx-qbit-test", tmpDir)
	defer cleanup()

	sc := NewStoreClient(&StoreClientConfig{})
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + nginxURL

	lockedLink := LockedFileLink("").create(testHash, files[0].Index)
	p := &store.GenerateLinkParams{Link: lockedLink}
	p.APIKey = token
	data, err := sc.GenerateLink(p)
	s.Require().NoError(err)

	resp, err := http.Get(data.Link)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode,
		"nginx should serve the file at the generated URL")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Equal(testContent, string(body))
}

func (s *QBitIntegrationSuite) TestH17_GenerateLink_NginxWithPathMapping() {
	files := s.waitForFiles(testHash)

	torrents, err := s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	s.Require().Len(torrents, 1)
	savePath := strings.TrimRight(torrents[0].SavePath, "/")

	tmpDir, err := os.MkdirTemp("", "qbit-nginx-pathmap-")
	s.Require().NoError(err)
	defer os.RemoveAll(tmpDir)

	testContent := "mapped-content-" + files[0].Name
	for _, f := range files {
		filePath := filepath.Join(tmpDir, "mapped", f.Name)
		s.Require().NoError(os.MkdirAll(filepath.Dir(filePath), 0755))
		s.Require().NoError(os.WriteFile(filePath, []byte("mapped-content-"+f.Name), 0644))
	}

	nginxURL, cleanup := s.startNginx("nginx-qbit-pathmap", tmpDir)
	defer cleanup()

	pathMap := savePath + ":/mapped"
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + nginxURL + "|" + pathMap

	sc := NewStoreClient(&StoreClientConfig{})
	lockedLink := LockedFileLink("").create(testHash, files[0].Index)
	p := &store.GenerateLinkParams{Link: lockedLink}
	p.APIKey = token
	data, err := sc.GenerateLink(p)
	s.Require().NoError(err)

	s.Contains(data.Link, "/mapped/")

	resp, err := http.Get(data.Link)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode,
		"nginx should serve the file at the mapped path")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Equal(testContent, string(body))
}

func (s *QBitIntegrationSuite) TestH2_GenerateLink_InvalidLink() {
	sc := NewStoreClient(&StoreClientConfig{})
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + s.cfg.FileBaseURL

	params := &store.GenerateLinkParams{
		Link: "not-a-valid-locked-link",
	}
	params.APIKey = token
	_, err := sc.GenerateLink(params)
	s.Error(err)
}

func (s *QBitIntegrationSuite) TestH3_GenerateLink_OutOfRange() {
	sc := NewStoreClient(&StoreClientConfig{})
	token := s.cfg.URL + "|" + s.cfg.Username + "|" + s.cfg.Password + "|" + s.cfg.FileBaseURL

	lockedLink := LockedFileLink("").create(testHash, 9999)
	params := &store.GenerateLinkParams{
		Link: lockedLink,
	}
	params.APIKey = token
	_, err := sc.GenerateLink(params)
	s.Error(err)
	s.Contains(err.Error(), "out of range")
}

func (s *QBitIntegrationSuite) TestI_DeleteTorrent() {
	err := s.client.DeleteTorrents(s.cfg, []string{testHash}, true)
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	torrents, err := s.client.GetTorrents(s.cfg, []string{testHash}, 0, 0)
	s.Require().NoError(err)
	s.Len(torrents, 0)
}

func TestQBitIntegration(t *testing.T) {
	suite.Run(t, new(QBitIntegrationSuite))
}
