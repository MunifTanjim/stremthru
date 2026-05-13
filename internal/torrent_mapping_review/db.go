package torrent_mapping_review

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "torrent_mapping_review"

type ColumnStruct struct {
	Id        string
	Hash      string
	Target    string
	Reason    string
	PrevId    string
	MappingId string
	Files     string
	Comment   string
	IP        string
	CreatedAt string
}

var Column = ColumnStruct{
	Id:        "id",
	Hash:      "hash",
	Target:    "target",
	Reason:    "reason",
	PrevId:    "prev_id",
	MappingId: "mapping_id",
	Files:     "files",
	Comment:   "comment",
	IP:        "ip",
	CreatedAt: "created_at",
}

type MappingTarget string

const (
	MappingTargetIMDB  MappingTarget = "imdb"
	MappingTargetAniDB MappingTarget = "anidb"
)

type ReviewReason string

const (
	ReviewReasonWrongMapping         ReviewReason = "wrong_mapping"
	ReviewReasonFakeTorrent          ReviewReason = "fake_torrent"
	ReviewReasonIncompleteSeasonPack ReviewReason = "incomplete_season_pack"
	ReviewReasonOther                ReviewReason = "other"
)

type FileCorrection struct {
	Path        string `json:"path"`
	PrevSeason  int    `json:"prev_season"`
	Season      int    `json:"season"`
	PrevEpisode int    `json:"prev_episode"`
	Episode     int    `json:"episode"`
}

type MappingReview struct {
	Id        int              `json:"id"`
	Hash      string           `json:"hash"`
	Target    MappingTarget    `json:"target"`
	Reason    ReviewReason     `json:"reason"`
	PrevId    string           `json:"prev_id"`
	MappingId string           `json:"mapping_id"`
	Files     []FileCorrection `json:"files"`
	Comment   string           `json:"comment"`
	IP        string           `json:"ip"`
	CreatedAt time.Time        `json:"created_at"`
}

var query_insert = fmt.Sprintf(
	"INSERT INTO %s (%s, %s, %s, %s, %s, %s, %s, %s) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	TableName,
	Column.Hash, Column.Target, Column.Reason, Column.PrevId,
	Column.MappingId, Column.Files, Column.Comment, Column.IP,
)

func Insert(item MappingReview) error {
	filesJSON, err := json.Marshal(item.Files)
	if err != nil {
		return err
	}

	_, err = db.Exec(query_insert, item.Hash, item.Target, item.Reason, item.PrevId,
		item.MappingId, string(filesJSON), item.Comment, item.IP)
	if err != nil {
		log.Error("failed to insert mapping review", "error", err)
		return err
	}
	return nil
}
