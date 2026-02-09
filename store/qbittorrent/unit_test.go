package qbittorrent

import (
	"net/http"
	"testing"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseToken ---

func TestParseToken_Valid(t *testing.T) {
	cfg, err := parseToken("http://localhost:8080|admin|password|http://fileserver")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.URL)
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "password", cfg.Password)
	assert.Equal(t, "http://fileserver", cfg.FileBaseURL)
}

func TestParseToken_TrailingSlashes(t *testing.T) {
	cfg, err := parseToken("http://localhost:8080/|admin|pass|http://fileserver/")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.URL)
	assert.Equal(t, "http://fileserver", cfg.FileBaseURL)
}

func TestParseToken_URLsWithColons(t *testing.T) {
	cfg, err := parseToken("https://seedbox.example.com:9443|user|p@ss:word|https://files.example.com:443/downloads")
	require.NoError(t, err)
	assert.Equal(t, "https://seedbox.example.com:9443", cfg.URL)
	assert.Equal(t, "user", cfg.Username)
	assert.Equal(t, "p@ss:word", cfg.Password)
	assert.Equal(t, "https://files.example.com:443/downloads", cfg.FileBaseURL)
}

func TestParseToken_TooFewParts(t *testing.T) {
	_, err := parseToken("http://localhost:8080|admin|password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 4 pipe-delimited parts")
}

func TestParseToken_WithPathMapping(t *testing.T) {
	cfg, err := parseToken("http://localhost:8080|admin|pass|http://server|/downloads:/media/torrents")
	require.NoError(t, err)
	assert.Equal(t, "http://server", cfg.FileBaseURL)
	assert.NotNil(t, cfg.PathMapping)
	assert.Equal(t, "/downloads", cfg.PathMapping.From)
	assert.Equal(t, "/media/torrents", cfg.PathMapping.To)
}

func TestParseToken_WithPathMapping_StripOnly(t *testing.T) {
	cfg, err := parseToken("http://localhost:8080|admin|pass|http://server|/downloads:")
	require.NoError(t, err)
	assert.NotNil(t, cfg.PathMapping)
	assert.Equal(t, "/downloads", cfg.PathMapping.From)
	assert.Equal(t, "", cfg.PathMapping.To)
}

func TestParseToken_WithoutPathMapping(t *testing.T) {
	cfg, err := parseToken("http://localhost:8080|admin|pass|http://server")
	require.NoError(t, err)
	assert.Nil(t, cfg.PathMapping)
}

func TestParseToken_EmptyPathMapping(t *testing.T) {
	// 5th field present but empty — no path mapping
	cfg, err := parseToken("http://localhost:8080|admin|pass|http://server|")
	require.NoError(t, err)
	assert.Nil(t, cfg.PathMapping)
}

func TestParseToken_InvalidPathMapping(t *testing.T) {
	_, err := parseToken("http://localhost:8080|admin|pass|http://server|no-colon")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from:to")
}

func TestParseToken_PathMappingEmptyFrom(t *testing.T) {
	_, err := parseToken("http://localhost:8080|admin|pass|http://server|:/external")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'from' is empty")
}

func TestParseToken_EmptyPart(t *testing.T) {
	_, err := parseToken("http://localhost:8080||password|http://fileserver")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "part 1 is empty")
}

func TestParseToken_EmptyString(t *testing.T) {
	_, err := parseToken("")
	assert.Error(t, err)
}

// --- progressToStatus ---

func TestProgressToStatus_Downloaded(t *testing.T) {
	assert.Equal(t, store.MagnetStatusDownloaded, progressToStatus(1.0))
	assert.Equal(t, store.MagnetStatusDownloaded, progressToStatus(1.1)) // edge: >1.0
}

func TestProgressToStatus_Downloading(t *testing.T) {
	assert.Equal(t, store.MagnetStatusDownloading, progressToStatus(0.5))
	assert.Equal(t, store.MagnetStatusDownloading, progressToStatus(0.001))
	assert.Equal(t, store.MagnetStatusDownloading, progressToStatus(0.999))
}

func TestProgressToStatus_Queued(t *testing.T) {
	assert.Equal(t, store.MagnetStatusQueued, progressToStatus(0))
	assert.Equal(t, store.MagnetStatusQueued, progressToStatus(-1)) // edge: negative
}

// --- LockedFileLink ---

func TestLockedFileLink_RoundTrip(t *testing.T) {
	link := LockedFileLink("")
	hash := "abcdef1234567890abcdef1234567890abcdef12"
	fileIndex := 3

	encoded := link.create(hash, fileIndex)
	assert.True(t, len(encoded) > len(lockedFileLinkPrefix))
	assert.Contains(t, encoded, lockedFileLinkPrefix)

	parsed := LockedFileLink(encoded)
	gotHash, gotIndex, err := parsed.parse()
	require.NoError(t, err)
	assert.Equal(t, hash, gotHash)
	assert.Equal(t, fileIndex, gotIndex)
}

func TestLockedFileLink_ZeroIndex(t *testing.T) {
	link := LockedFileLink("")
	encoded := link.create("abc123", 0)

	parsed := LockedFileLink(encoded)
	hash, idx, err := parsed.parse()
	require.NoError(t, err)
	assert.Equal(t, "abc123", hash)
	assert.Equal(t, 0, idx)
}

func TestLockedFileLink_LargeIndex(t *testing.T) {
	link := LockedFileLink("")
	encoded := link.create("abc123", 999)

	parsed := LockedFileLink(encoded)
	_, idx, err := parsed.parse()
	require.NoError(t, err)
	assert.Equal(t, 999, idx)
}

func TestLockedFileLink_InvalidData(t *testing.T) {
	parsed := LockedFileLink(lockedFileLinkPrefix + "not-valid-base64!!!")
	_, _, err := parsed.parse()
	assert.Error(t, err)
}

// --- TranslateStatusCode ---

func TestTranslateStatusCode(t *testing.T) {
	assert.Equal(t, core.ErrorCodeUnauthorized, TranslateStatusCode(http.StatusForbidden))
	assert.Equal(t, core.ErrorCodeNotFound, TranslateStatusCode(http.StatusNotFound))
	assert.Equal(t, core.ErrorCodeConflict, TranslateStatusCode(http.StatusConflict))
	assert.Equal(t, core.ErrorCodeServiceUnavailable, TranslateStatusCode(500))
	assert.Equal(t, core.ErrorCodeServiceUnavailable, TranslateStatusCode(503))
	assert.Equal(t, core.ErrorCodeBadRequest, TranslateStatusCode(400))
	assert.Equal(t, core.ErrorCodeBadRequest, TranslateStatusCode(422))
	assert.Equal(t, core.ErrorCodeUnknown, TranslateStatusCode(200))
	assert.Equal(t, core.ErrorCodeUnknown, TranslateStatusCode(301))
}

// --- QbitError ---

func TestQbitError_ErrorJSON(t *testing.T) {
	e := newQbitError(403, []byte("Forbidden"))
	assert.Contains(t, e.Error(), "403")
	assert.Contains(t, e.Error(), "Forbidden")
}

func TestQbitError_EmptyBody(t *testing.T) {
	e := newQbitError(500, []byte(""))
	assert.Equal(t, "Internal Server Error", e.Message)
}

// --- TorrentInfo ---

func TestTorrentInfo_GetAddedAt(t *testing.T) {
	ti := TorrentInfo{AddedOn: 1700000000}
	addedAt := ti.GetAddedAt()
	assert.Equal(t, int64(1700000000), addedAt.Unix())

	ti2 := TorrentInfo{AddedOn: 0}
	assert.Equal(t, int64(0), ti2.GetAddedAt().Unix())

	ti3 := TorrentInfo{AddedOn: -1}
	assert.Equal(t, int64(0), ti3.GetAddedAt().Unix())
}

// --- TorrentFile ---

func TestTorrentFile_GetName(t *testing.T) {
	f := TorrentFile{Name: "Ubuntu/ubuntu-22.04.iso"}
	assert.Equal(t, "ubuntu-22.04.iso", f.GetName())

	f2 := TorrentFile{Name: "single-file.mkv"}
	assert.Equal(t, "single-file.mkv", f2.GetName())
}

func TestTorrentFile_GetPath(t *testing.T) {
	f := TorrentFile{Name: "Ubuntu/ubuntu-22.04.iso"}
	assert.Equal(t, "/ubuntu-22.04.iso", f.GetPath())

	f2 := TorrentFile{Name: "FolderName/sub/deep/file.mkv"}
	assert.Equal(t, "/sub/deep/file.mkv", f2.GetPath())

	f3 := TorrentFile{Name: "single-file.mkv"}
	assert.Equal(t, "/single-file.mkv", f3.GetPath())
}

// --- UpstreamErrorWithCause ---

func TestUpstreamErrorWithCause_QbitError(t *testing.T) {
	qerr := newQbitError(403, []byte("Forbidden"))
	uerr := UpstreamErrorWithCause(qerr)
	assert.Equal(t, string(store.StoreNameQBittorrent), uerr.StoreName)
	assert.Equal(t, core.ErrorCodeUnauthorized, uerr.Code)
	assert.Equal(t, 403, uerr.StatusCode)
}

func TestUpstreamErrorWithCause_GenericError(t *testing.T) {
	err := UpstreamErrorWithCause(assert.AnError)
	assert.Equal(t, string(store.StoreNameQBittorrent), err.StoreName)
	assert.Equal(t, assert.AnError, err.Cause)
}

// --- pathMapping.apply ---

func TestPathMapping_BasicReplace(t *testing.T) {
	pm := &pathMapping{From: "/downloads", To: "/media/torrents"}
	assert.Equal(t, "/media/torrents/Movie/file.mkv", pm.apply("/downloads/Movie/file.mkv"))
}

func TestPathMapping_StripPrefix(t *testing.T) {
	pm := &pathMapping{From: "/downloads", To: ""}
	assert.Equal(t, "/Movie/file.mkv", pm.apply("/downloads/Movie/file.mkv"))
}

func TestPathMapping_NoMatch(t *testing.T) {
	pm := &pathMapping{From: "/downloads", To: "/media"}
	// Prefix doesn't match — path returned unchanged
	assert.Equal(t, "/other/path/file.mkv", pm.apply("/other/path/file.mkv"))
}

func TestPathMapping_TrailingSlashes(t *testing.T) {
	pm := &pathMapping{From: "/downloads/", To: "/media/"}
	assert.Equal(t, "/media/Movie/file.mkv", pm.apply("/downloads/Movie/file.mkv"))
}

func TestPathMapping_NestedFrom(t *testing.T) {
	pm := &pathMapping{From: "/data/downloads/complete", To: "/torrents"}
	assert.Equal(t, "/torrents/Movie/file.mkv", pm.apply("/data/downloads/complete/Movie/file.mkv"))
}

func TestPathMapping_PartialPrefixNoMatch(t *testing.T) {
	// "/downloads-extra" should NOT match "/downloads"
	pm := &pathMapping{From: "/downloads", To: "/media"}
	assert.Equal(t, "/downloads-extra/file.mkv", pm.apply("/downloads-extra/file.mkv"))
}

func TestPathMapping_ExactMatch(t *testing.T) {
	pm := &pathMapping{From: "/downloads", To: "/media"}
	assert.Equal(t, "/media", pm.apply("/downloads"))
}

// --- buildFileURL ---

func TestBuildFileURL_SimpleName(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "ubuntu-22.04.iso")
	assert.Equal(t, "http://localhost:8080/ubuntu-22.04.iso", link)
}

func TestBuildFileURL_NestedPath(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "Ubuntu 22.04/ubuntu-22.04.iso")
	assert.Equal(t, "http://localhost:8080/Ubuntu%2022.04/ubuntu-22.04.iso", link)
}

func TestBuildFileURL_SpecialCharacters(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "Movie (2024)/Movie [1080p] (2024).mkv")
	assert.Equal(t, "http://localhost:8080/Movie%20%282024%29/Movie%20%5B1080p%5D%20%282024%29.mkv", link)
}

func TestBuildFileURL_Unicode(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "映画/テスト.mkv")
	assert.Contains(t, link, "http://localhost:8080/")
	assert.NotContains(t, link, "映画")  // unicode should be percent-encoded
	assert.NotContains(t, link, "テスト") // unicode should be percent-encoded
}

func TestBuildFileURL_DeeplyNested(t *testing.T) {
	link := buildFileURL("http://files.example.com", "Show/Season 1/Episode 01.mkv")
	assert.Equal(t, "http://files.example.com/Show/Season%201/Episode%2001.mkv", link)
}

func TestBuildFileURL_PreservesSlashes(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "folder/sub/file.mkv")
	assert.Equal(t, "http://localhost:8080/folder/sub/file.mkv", link)
}

func TestBuildFileURL_BaseURLWithPath(t *testing.T) {
	link := buildFileURL("http://files.example.com/downloads", "movie.mkv")
	assert.Equal(t, "http://files.example.com/downloads/movie.mkv", link)
}

func TestBuildFileURL_HashAndPercent(t *testing.T) {
	link := buildFileURL("http://localhost:8080", "file #1 100%.mkv")
	assert.Contains(t, link, "%23")   // # encoded
	assert.Contains(t, link, "%25")   // % encoded
	assert.NotContains(t, link, " #") // space and # should be encoded
}

// --- getConfig ---

func TestGetConfig_EmptyKey(t *testing.T) {
	sc := NewStoreClient(&StoreClientConfig{})
	_, err := sc.getConfig("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing api key")
}

func TestGetConfig_ValidKey(t *testing.T) {
	sc := NewStoreClient(&StoreClientConfig{})
	cfg, err := sc.getConfig("http://localhost:8080|admin|pass|http://files:8080")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.URL)
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "pass", cfg.Password)
	assert.Equal(t, "http://files:8080", cfg.FileBaseURL)
}

func TestGetConfig_InvalidKey(t *testing.T) {
	sc := NewStoreClient(&StoreClientConfig{})
	_, err := sc.getConfig("not-a-valid-token")
	assert.Error(t, err)
}
