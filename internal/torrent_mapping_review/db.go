package torrent_mapping_review

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "torrent_mapping_review"

type ColumnStruct struct {
	Id         string
	Hash       string
	Target     string
	Reason     string
	PrevId     string
	MappingId  string
	Files      string
	Comment    string
	IP         string
	CreatedAt  string
	Status            string
	ResolvedAt        string
	SuggestedMappings string
}

var Column = ColumnStruct{
	Id:         "id",
	Hash:       "hash",
	Target:     "target",
	Reason:     "reason",
	PrevId:     "prev_id",
	MappingId:  "mapping_id",
	Files:      "files",
	Comment:    "comment",
	IP:         "ip",
	CreatedAt:  "created_at",
	Status:            "status",
	ResolvedAt:        "resolved_at",
	SuggestedMappings: "suggested_mappings",
}

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusResolved ReviewStatus = "resolved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

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

type SuggestedMapping struct {
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}

type MappingReview struct {
	Id         int              `json:"id"`
	Hash       string           `json:"hash"`
	Target     MappingTarget    `json:"target"`
	Reason     ReviewReason     `json:"reason"`
	PrevId     string           `json:"prev_id"`
	MappingId  string           `json:"mapping_id"`
	Files             []FileCorrection   `json:"files"`
	SuggestedMappings []SuggestedMapping `json:"suggested_mappings,omitempty"`
	Comment           string             `json:"comment"`
	IP         string           `json:"ip"`
	CreatedAt  time.Time        `json:"created_at"`
	Status     ReviewStatus     `json:"status"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`
}

var columns = []string{
	Column.Id,
	Column.Hash,
	Column.Target,
	Column.Reason,
	Column.PrevId,
	Column.MappingId,
	Column.Files,
	Column.Comment,
	Column.IP,
	Column.CreatedAt,
	Column.Status,
	Column.ResolvedAt,
	Column.SuggestedMappings,
}

var query_insert = fmt.Sprintf(
	"INSERT INTO %s (%s, %s, %s, %s, %s, %s, %s, %s, %s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	TableName,
	Column.Hash, Column.Target, Column.Reason, Column.PrevId,
	Column.MappingId, Column.Files, Column.Comment, Column.IP, Column.SuggestedMappings,
)

func Insert(item MappingReview) error {
	filesJSON, err := json.Marshal(item.Files)
	if err != nil {
		return err
	}

	suggestedJSON, err := json.Marshal(item.SuggestedMappings)
	if err != nil {
		return err
	}

	_, err = db.Exec(query_insert, item.Hash, item.Target, item.Reason, item.PrevId,
		item.MappingId, string(filesJSON), item.Comment, item.IP, string(suggestedJSON))
	if err != nil {
		log.Error("failed to insert mapping review", "error", err)
		return err
	}
	return nil
}

func scanMappingReview(scanner interface {
	Scan(dest ...any) error
}) (MappingReview, error) {
	var item MappingReview
	var filesJSON string
	var suggestedJSON sql.NullString
	var createdAt db.Timestamp
	var resolvedAt db.Timestamp
	if err := scanner.Scan(
		&item.Id,
		&item.Hash,
		&item.Target,
		&item.Reason,
		&item.PrevId,
		&item.MappingId,
		&filesJSON,
		&item.Comment,
		&item.IP,
		&createdAt,
		&item.Status,
		&resolvedAt,
		&suggestedJSON,
	); err != nil {
		return item, err
	}
	item.CreatedAt = createdAt.Time
	if !resolvedAt.IsZero() {
		t := resolvedAt.Time
		item.ResolvedAt = &t
	}
	if filesJSON != "" && filesJSON != "null" {
		if err := json.Unmarshal([]byte(filesJSON), &item.Files); err != nil {
			return item, err
		}
	}
	if item.Files == nil {
		item.Files = []FileCorrection{}
	}
	if suggestedJSON.Valid && suggestedJSON.String != "" && suggestedJSON.String != "null" {
		if err := json.Unmarshal([]byte(suggestedJSON.String), &item.SuggestedMappings); err != nil {
			return item, err
		}
	}
	if item.SuggestedMappings == nil {
		item.SuggestedMappings = []SuggestedMapping{}
	}
	return item, nil
}

type ListParams struct {
	Status string
	Target string
	Cursor string
	Limit  int
}

type ListResult struct {
	Items      []MappingReview
	NextCursor string
}

func List(params ListParams) (ListResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	var query strings.Builder
	query.WriteString(fmt.Sprintf("SELECT %s FROM %s", db.JoinColumnNames(columns...), TableName))

	args := []any{}
	conditions := []string{}

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("%s = ?", Column.Status))
		args = append(args, params.Status)
	}
	if params.Target != "" {
		conditions = append(conditions, fmt.Sprintf("%s = ?", Column.Target))
		args = append(args, params.Target)
	}
	if params.Cursor != "" {
		cursorId, err := strconv.Atoi(params.Cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("%s < ?", Column.Id))
			args = append(args, cursorId)
		}
	}

	if len(conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(conditions, " AND "))
	}

	query.WriteString(fmt.Sprintf(" ORDER BY %s DESC LIMIT ?", Column.Id))
	args = append(args, limit+1)

	rows, err := db.Query(query.String(), args...)
	if err != nil {
		log.Error("failed to list mapping reviews", "error", err)
		return ListResult{}, err
	}
	defer rows.Close()

	items := []MappingReview{}
	for rows.Next() {
		item, err := scanMappingReview(rows)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	result := ListResult{Items: items}
	if len(items) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.Itoa(result.Items[limit-1].Id)
	}
	return result, nil
}

var query_get_by_id = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ?",
	db.JoinColumnNames(columns...),
	TableName,
	Column.Id,
)

func GetById(id int) (*MappingReview, error) {
	row := db.QueryRow(query_get_by_id, id)
	item, err := scanMappingReview(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_resolve = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = %s WHERE %s = ?",
	TableName,
	Column.Status,
	Column.ResolvedAt,
	db.CurrentTimestamp,
	Column.Id,
)

func Resolve(id int) error {
	_, err := db.Exec(query_resolve, ReviewStatusResolved, id)
	if err != nil {
		log.Error("failed to resolve mapping review", "error", err, "id", id)
	}
	return err
}

var query_reject = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = %s WHERE %s = ?",
	TableName,
	Column.Status,
	Column.ResolvedAt,
	db.CurrentTimestamp,
	Column.Id,
)

func Reject(id int) error {
	_, err := db.Exec(query_reject, ReviewStatusRejected, id)
	if err != nil {
		log.Error("failed to reject mapping review", "error", err, "id", id)
	}
	return err
}
