package buddy

import (
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/peer"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tss "github.com/MunifTanjim/stremthru/internal/torrent_stream/torrent_stream_syncinfo"
)

var PullPeer, pullLocalOnly = func() (*peer.APIClient, bool) {
	baseUrl := config.PullPeerURL
	if baseUrl == "" {
		baseUrl = config.PeerURL
	}
	localOnly := baseUrl == config.PullPeerURL
	if baseUrl == "" {
		return nil, localOnly
	}
	return peer.NewAPIClient(&peer.APIClientConfig{
		BaseURL: baseUrl,
	}), localOnly
}()

var pullPeerLog = logger.Scoped("peer:pull")

var noTorrentInfo = !config.Feature.HasTorrentInfo()

// supports imdb or anidb
func PullTorrentsByStremId(sid string, originInstanceId string) {
	if noTorrentInfo || PullPeer == nil || PullPeer.IsHaltedCheckMagnet() || !tss.ShouldPull(sid) {
		return
	}

	cleanSId := ts.CleanStremId(sid)
	start := time.Now()
	res, err := PullPeer.ListTorrents(&peer.ListTorrentsByStremIdParams{
		SId:              cleanSId,
		LocalOnly:        pullLocalOnly,
		OriginInstanceId: originInstanceId,
	})
	duration := time.Since(start)

	if err != nil {
		if duration > 25*time.Second {
			PullPeer.HaltCheckMagnet()
		}

		pullPeerLog.Error("failed to pull torrents", "error", core.PackError(err), "duration", duration, "sid", cleanSId)
		return
	}

	count := len(res.Data.Items)
	pullPeerLog.Info("pulled torrents", "duration", duration, "sid", cleanSId, "count", count)

	items := make([]ti.TorrentInfoInsertData, count)
	for i := range res.Data.Items {
		data := &res.Data.Items[i]
		items[i] = ti.TorrentInfoInsertData{
			Hash:         data.Hash,
			TorrentTitle: data.TorrentTitle,
			Size:         data.Size,
			Source:       ti.TorrentInfoSource(data.Source),
			Category:     ti.TorrentInfoCategory(data.Category),
			Files:        data.Files,
			Seeders:      data.Seeders,
			Leechers:     data.Leechers,
			Private:      data.Private,
		}
	}
	ti.Upsert(items, "", false)
	go tss.MarkPulled(cleanSId)
}

func ListTorrentsByStremId(sid string, localOnly bool, originInstanceId string, noMissingSize bool) (*ti.ListTorrentsData, error) {
	if originInstanceId == config.InstanceId && !pullLocalOnly {
		pullPeerLog.Info("loop detected for list torrents, self-correcting...")
		pullLocalOnly = true
	}

	if !localOnly {
		PullTorrentsByStremId(sid, originInstanceId)
	}

	data, err := ti.ListByStremId(sid, noMissingSize)
	if err != nil {
		return nil, err
	}
	return data, nil
}
