package store_util

import (
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/store"
)

func RecordTorrentInfoFromListMagnets(storeCode store.StoreCode, items []store.ListMagnetsDataItem) {
	switch storeCode {
	case store.StoreCodeAllDebrid, store.StoreCodeDebridLink, store.StoreCodeRealDebrid, store.StoreCodeTorBox:
		break
	default:
		return
	}

	upsertItems := []ti.TorrentInfoInsertData{}
	for i := range items {
		item := &items[i]
		if item.Name == "" {
			continue
		}
		upsertItems = append(upsertItems, ti.TorrentInfoInsertData{
			Hash:         item.Hash,
			TorrentTitle: item.Name,
			Size:         item.Size,
			Source:       ti.TorrentInfoSource(storeCode),
		})
		ti.Upsert(upsertItems, ti.TorrentInfoCategoryUnknown, storeCode != store.StoreCodeRealDebrid)
	}
}
