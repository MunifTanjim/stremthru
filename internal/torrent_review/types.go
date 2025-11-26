package torrent_review

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type ReviewReason string

const (
	ReviewReasonWrongMapping         ReviewReason = "wrong_mapping"
	ReviewReasonFakeTorrent          ReviewReason = "fake_torrent"
	ReviewReasonIncompleteSeasonPack ReviewReason = "incomplete_season_pack"
	ReviewReasonWrongTitle           ReviewReason = "wrong_title"
	ReviewReasonOther                ReviewReason = "other"
)

func (r ReviewReason) String() string {
	return string(r)
}

func (r ReviewReason) IsValid() bool {
	switch r {
	case ReviewReasonWrongMapping,
		ReviewReasonFakeTorrent,
		ReviewReasonIncompleteSeasonPack,
		ReviewReasonWrongTitle,
		ReviewReasonOther:
		return true
	}
	return false
}

type File struct {
	Path        string `json:"path"`
	PrevSeason  int    `json:"prev_s"`
	Season      int    `json:"s"`
	PrevEpisode int    `json:"prev_ep"`
	Episode     int    `json:"ep"`
}

type Files []File

func (files Files) Value() (driver.Value, error) {
	return json.Marshal(files)
}

func (files *Files) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, files)
}
