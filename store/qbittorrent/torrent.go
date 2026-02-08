package qbittorrent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type TorrentInfo struct {
	AddedOn      int64   `json:"added_on"`
	AmountLeft   int64   `json:"amount_left"`
	Category     string  `json:"category"`
	Completed    int64   `json:"completed"`
	CompletionOn int64   `json:"completion_on"`
	ContentPath  string  `json:"content_path"`
	DlSpeed      int64   `json:"dl_speed"`
	Downloaded   int64   `json:"downloaded"`
	ETA          int64   `json:"eta"`
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	NumComplete  int     `json:"num_complete"`
	NumSeeds     int     `json:"num_seeds"`
	Priority     int     `json:"priority"`
	Progress     float64 `json:"progress"`
	SavePath     string  `json:"save_path"`
	SeqDl        bool    `json:"seq_dl"`
	Size         int64   `json:"size"`
	State        string  `json:"state"`
	TotalSize    int64   `json:"total_size"`
	UpSpeed      int64   `json:"up_speed"`
	Uploaded     int64   `json:"uploaded"`
}

func (t *TorrentInfo) GetAddedAt() time.Time {
	if t.AddedOn <= 0 {
		return time.Unix(0, 0).UTC()
	}
	return time.Unix(t.AddedOn, 0).UTC()
}

type TorrentFile struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
	Priority int     `json:"priority"`
}

func (f *TorrentFile) GetName() string {
	return filepath.Base(f.Name)
}

func (f *TorrentFile) GetPath() string {
	parts := strings.SplitN(strings.TrimPrefix(f.Name, "/"), "/", 2)
	if len(parts) == 2 {
		return "/" + parts[1]
	}
	return "/" + f.Name
}

// GetVersion calls GET /api/v2/app/version
func (c *APIClient) GetVersion(cfg *qbitConfig) (string, error) {
	resp, body, err := c.doRequest(cfg, "GET", "/api/v2/app/version", nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", newQbitError(resp.StatusCode, body)
	}
	return strings.TrimSpace(string(body)), nil
}

// GetTorrents calls GET /api/v2/torrents/info.
// If hashes is non-empty, filters by those hashes (pipe-separated).
func (c *APIClient) GetTorrents(cfg *qbitConfig, hashes []string, limit, offset int) ([]TorrentInfo, error) {
	form := url.Values{}
	if len(hashes) > 0 {
		form.Set("hashes", strings.Join(hashes, "|"))
	}
	if limit > 0 {
		form.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		form.Set("offset", fmt.Sprintf("%d", offset))
	}
	resp, body, err := c.doRequest(cfg, "GET", "/api/v2/torrents/info", form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, newQbitError(resp.StatusCode, body)
	}
	var torrents []TorrentInfo
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

// GetTorrentFiles calls GET /api/v2/torrents/files
func (c *APIClient) GetTorrentFiles(cfg *qbitConfig, hash string) ([]TorrentFile, error) {
	form := url.Values{"hash": {hash}}
	resp, body, err := c.doRequest(cfg, "GET", "/api/v2/torrents/files", form)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, newQbitError(resp.StatusCode, body)
	}
	var files []TorrentFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// AddTorrentMagnet calls POST /api/v2/torrents/add with a magnet URI.
// Enables sequential download and first/last piece priority for streaming.
func (c *APIClient) AddTorrentMagnet(cfg *qbitConfig, magnetURI string) error {
	form := url.Values{
		"urls":               {magnetURI},
		"sequentialDownload": {"true"},
		"firstLastPiecePrio": {"true"},
	}
	resp, body, err := c.doRequest(cfg, "POST", "/api/v2/torrents/add", form)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newQbitError(resp.StatusCode, body)
	}
	return nil
}

// DeleteTorrents calls POST /api/v2/torrents/delete.
// deleteFiles controls whether downloaded data is also removed from disk.
func (c *APIClient) DeleteTorrents(cfg *qbitConfig, hashes []string, deleteFiles bool) error {
	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}
	form := url.Values{
		"hashes":      {strings.Join(hashes, "|")},
		"deleteFiles": {deleteFilesStr},
	}
	resp, body, err := c.doRequest(cfg, "POST", "/api/v2/torrents/delete", form)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newQbitError(resp.StatusCode, body)
	}
	return nil
}
