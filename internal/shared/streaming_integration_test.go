//go:build integration

package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/store/qbittorrent"
)

// Big Buck Bunny: public domain, multi-file, has web seeds for fast download.
const bbbMagnet = "magnet:?xt=urn:btih:dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c&dn=Big+Buck+Bunny&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fbig-buck-bunny.torrent"
const bbbHash = "dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c"

const qbitContainerName = "qbit-streaming-integ-test"

// TestFFProbeWithRealProxy verifies that ffprobe can probe a partially
// downloaded video file served through the real ProxyResponse function.
//
// This is the end-to-end streaming scenario:
//   - qBittorrent downloads with firstLastPiecePrio (first + last pieces first)
//   - A Go file server simulates nginx serving the pre-allocated file
//   - ProxyResponse sits in front with SafeBytesFunc + IsRangeAvailableFunc
//   - ffprobe reads start (MP4 header) + seeks to end (moov atom) — both available
func TestFFProbeWithRealProxy(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed, skipping")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed, skipping")
	}

	// --- Start qBittorrent container ---
	port, err := qbittorrent.FreePort()
	if err != nil {
		t.Fatalf("could not find free port: %v", err)
	}

	downloadDir, err := os.MkdirTemp("", "qbit-streaming-test-")
	if err != nil {
		t.Fatalf("could not create temp dir: %v", err)
	}
	defer os.RemoveAll(downloadDir)

	_ = exec.Command("docker", "rm", "-f", qbitContainerName).Run()
	cmd := exec.Command("docker", "run", "-d", "--name", qbitContainerName,
		"-p", fmt.Sprintf("%d:%d", port, port),
		"-e", fmt.Sprintf("WEBUI_PORT=%d", port),
		"-e", "PUID=1000", "-e", "PGID=1000",
		"-v", downloadDir+":/downloads",
		"lscr.io/linuxserver/qbittorrent:latest",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to start qBit container: %s: %v", string(out), err)
	}
	defer exec.Command("docker", "rm", "-f", qbitContainerName).Run()

	qbitURL := fmt.Sprintf("http://localhost:%d", port)
	if err := qbittorrent.WaitForHTTP(qbitURL, 30*time.Second); err != nil {
		t.Fatalf("qBit WebUI not ready: %v", err)
	}

	pass, err := qbittorrent.GetQbitPassword(qbitContainerName, 30*time.Second)
	if err != nil {
		t.Fatalf("could not get qBit password: %v", err)
	}
	t.Logf("qBit ready on port %d (password: %s)", port, pass)

	// --- Set up qBit API client ---
	apiClient := qbittorrent.NewAPIClientForTest()
	cfg := qbittorrent.NewConfigForTest(qbitURL, "admin", pass, qbitURL)

	// --- Add BBB torrent ---
	if err := apiClient.AddTorrentMagnet(cfg, bbbMagnet); err != nil {
		t.Fatalf("failed to add torrent: %v", err)
	}
	defer apiClient.DeleteTorrents(cfg, []string{bbbHash}, true)

	// Wait for metadata
	var files []qbittorrent.TorrentFile
	for i := 0; i < 30; i++ {
		files, err = apiClient.GetTorrentFiles(cfg, bbbHash)
		if err == nil && len(files) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if len(files) == 0 {
		t.Fatal("timed out waiting for torrent files")
	}

	// Find the video file (largest)
	var video *qbittorrent.TorrentFile
	for i := range files {
		if video == nil || files[i].Size > video.Size {
			video = &files[i]
		}
	}
	t.Logf("Video: %s (%d MB), pieces [%d, %d]",
		video.Name, video.Size/1024/1024, video.PieceRange[0], video.PieceRange[1])

	// Wait for first + last pieces
	fp, lp := video.PieceRange[0], video.PieceRange[1]
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		states, err := apiClient.GetPieceStates(cfg, bbbHash)
		if err == nil && lp < len(states) && states[fp] == 2 && states[lp] == 2 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Pause to freeze download state
	if err := apiClient.PauseTorrents(cfg, []string{bbbHash}); err != nil {
		t.Fatalf("failed to pause: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Verify the file on disk
	filePath := filepath.Join(downloadDir, video.Name)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("video file not on disk: %s: %v", filePath, err)
	}
	t.Logf("File on disk: %s (%d MB)", filePath, fileInfo.Size()/1024/1024)

	// --- Set up the qBit store client for callbacks ---
	sc := qbittorrent.NewStoreClient(&qbittorrent.StoreClientConfig{})
	token := qbitURL + "|admin|" + pass + "|" + qbitURL

	// SafeBytesFunc: reports contiguous bytes downloaded from start of file
	safeBytesFn := SafeBytesFunc(func() (int64, int64, bool) {
		safe, fileSize, done, err := sc.GetSafeBytes(token, bbbHash, video.Index)
		if err != nil {
			return 0, 0, true // assume done on error
		}
		return safe, fileSize, done
	})

	// IsRangeAvailableFunc: piece-level check for non-contiguous ranges
	isRangeAvailFn := IsRangeAvailableFunc(func(start, end int64) bool {
		avail, err := sc.IsFileRangeAvailable(token, bbbHash, video.Index, start, end)
		if err != nil {
			t.Logf("[proxy] range check error: %v", err)
			return false
		}
		if avail {
			t.Logf("[proxy] range [%d, %d] verified available at piece level", start, end)
		}
		return avail
	})

	// --- Start file server (simulates nginx) ---
	fileServer := httptest.NewServer(http.FileServer(http.Dir(downloadDir)))
	defer fileServer.Close()

	// --- Start proxy server using real ProxyResponse ---
	var servedRequests []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		servedRequests = append(servedRequests, r.Method+" "+r.Header.Get("Range"))
		// Build upstream URL (file server)
		upstream := fileServer.URL + r.URL.Path
		ProxyResponse(w, r, upstream, config.TUNNEL_TYPE_NONE, safeBytesFn, isRangeAvailFn)
	}))
	defer proxy.Close()

	// --- Run ffprobe ---
	// The video file path inside the torrent is "Big Buck Bunny/Big Buck Bunny.mp4"
	probeURL := proxy.URL + "/" + video.Name
	t.Logf("Running ffprobe against: %s", probeURL)

	ffprobe := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		probeURL,
	)
	output, err := ffprobe.Output()

	t.Logf("Requests: %v", servedRequests)
	if err != nil {
		t.Fatalf("ffprobe failed: %v\noutput: %s", err, string(output))
	}

	// --- Verify ffprobe results ---
	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse ffprobe JSON: %v", err)
	}

	if result.Format.Duration == "" {
		t.Error("ffprobe should detect duration")
	}
	t.Logf("Duration: %s seconds", result.Format.Duration)

	hasVideo := false
	for _, s := range result.Streams {
		if s.CodecType == "video" {
			hasVideo = true
			t.Logf("Video stream: %s %dx%d", s.CodecName, s.Width, s.Height)
			if s.Width == 0 {
				t.Error("should detect video width")
			}
		}
	}
	if !hasVideo {
		t.Error("ffprobe should find a video stream")
	}
}
