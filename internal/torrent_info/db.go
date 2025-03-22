package torrent_info

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type CommaSeperatedString []string

func (css CommaSeperatedString) Value() (driver.Value, error) {
	return strings.Join(css, ","), nil
}

func (css *CommaSeperatedString) Scan(value any) error {
	if value == nil {
		*css = []string{}
		return nil
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New("failed to convert value to string")
	}
	if str == "" {
		*css = []string{}
		return nil
	}
	*css = strings.Split(str, ",")
	return nil
}

type CommaSeperatedInt []int

func (csi CommaSeperatedInt) Value() (driver.Value, error) {
	css := make(CommaSeperatedString, len(csi))
	for i := range csi {
		css[i] = strconv.Itoa(csi[i])
	}
	return css.Value()
}

func (csi *CommaSeperatedInt) Scan(value any) error {
	css := CommaSeperatedString{}
	if err := css.Scan(value); err != nil {
		return err
	}
	*csi = make([]int, len(css))
	for i := range css {
		v, err := strconv.Atoi(css[i])
		if err != nil {
			return err
		}
		(*csi)[i] = v
	}
	return nil
}

type TorrentInfoSource string

const (
	TorrentInfoSourceTorrentio TorrentInfoSource = "tio"
)

type TorrentInfoCategory string

const (
	TorrentInfoCategoryMovie   TorrentInfoCategory = "movie"
	TorrentInfoCategorySeries  TorrentInfoCategory = "series"
	TorrentInfoCategoryXXX     TorrentInfoCategory = "xxx"
	TorrentInfoCategoryUnknown TorrentInfoCategory = ""
)

type TorrentInfo struct {
	Hash         string `json:"hash"`
	TorrentTitle string `json:"t_title"`

	Source   string              `json:"src"`
	Category TorrentInfoCategory `json:"category"`
	ImdbId   string              `json:"imdb_id"`

	CreatedAt     db.Timestamp `json:"created_at"`
	UpdatedAt     db.Timestamp `json:"updated_at"`
	ParsedAt      db.Timestamp `json:"parsed_at"`
	ParserVersion int          `json:"parser_version"`
	SentAt        db.Timestamp `json:"sent_at,omitzero"`

	Audio        CommaSeperatedString `json:"audio"`
	BitDepth     string               `json:"bit_depth"`
	Channel      string               `json:"channel"`
	Codec        string               `json:"codec"`
	Complete     bool                 `json:"complete"`
	Container    string               `json:"container"`
	Date         string               `json:"date"`
	Documentary  bool                 `json:"documentary"`
	Dubbed       bool                 `json:"dubbed"`
	Edition      string               `json:"edition"`
	Episodes     CommaSeperatedInt    `json:"episodes"`
	Extension    string               `json:"extension"`
	ReleaseGroup string               `json:"release_group"`
	HDR          CommaSeperatedString `json:"hdr"`
	Hardcoded    bool                 `json:"hardcoded"`
	Is3D         bool                 `json:"is_3d"`
	Languages    CommaSeperatedString `json:"languages"`
	Network      string               `json:"network"`
	Proper       bool                 `json:"proper"`
	Quality      string               `json:"quality"`
	Remastered   bool                 `json:"remastered"`
	Repack       bool                 `json:"repack"`
	Resolution   string               `json:"resolution"`
	Retail       bool                 `json:"retail"`
	Seasons      CommaSeperatedInt    `json:"seasons"`
	Site         string               `json:"site"`
	Size         int64                `json:"size"`
	Subbed       bool                 `json:"subbed"`
	Title        string               `json:"title"`
	Unrated      bool                 `json:"unrated"`
	Upscaled     bool                 `json:"upscaled"`
	Volumes      CommaSeperatedInt    `json:"volumes"`
	Year         int                  `json:"year"`
}

const TableName = "torrent_info"

type ColumnStruct struct {
	Hash         string
	TorrentTitle string

	Source        string
	Category      string
	ImdbId        string
	CreatedAt     string
	UpdatedAt     string
	ParsedAt      string
	ParserVersion string
	SentAt        string

	Audio        string
	BitDepth     string
	Channel      string
	Codec        string
	Complete     string
	Container    string
	Date         string
	Documentary  string
	Dubbed       string
	Edition      string
	Episodes     string
	Extension    string
	ReleaseGroup string
	HDR          string
	Hardcoded    string
	Is3D         string
	Languages    string
	Network      string
	Proper       string
	Quality      string
	Remastered   string
	Repack       string
	Resolution   string
	Retail       string
	Seasons      string
	Site         string
	Size         string
	Subbed       string
	Title        string
	Unrated      string
	Upscaled     string
	Volumes      string
	Year         string
}

var Column = ColumnStruct{
	Hash:         "hash",
	TorrentTitle: "t_title",

	Source:        "src",
	Category:      "category",
	ImdbId:        "imdb_id",
	CreatedAt:     "created_at",
	UpdatedAt:     "updated_at",
	ParsedAt:      "parsed_at",
	ParserVersion: "parser_version",
	SentAt:        "sent_at",

	Audio:        "audio",
	BitDepth:     "bit_depth",
	Channel:      "channel",
	Codec:        "codec",
	Complete:     "complete",
	Container:    "container",
	Date:         "date",
	Documentary:  "documentary",
	Dubbed:       "dubbed",
	Edition:      "edition",
	Episodes:     "episodes",
	Extension:    "extension",
	ReleaseGroup: "release_group",
	HDR:          "hdr",
	Hardcoded:    "hardcoded",
	Is3D:         "is_3d",
	Languages:    "languages",
	Network:      "network",
	Proper:       "proper",
	Quality:      "quality",
	Remastered:   "remastered",
	Repack:       "repack",
	Resolution:   "resolution",
	Retail:       "retail",
	Seasons:      "seasons",
	Site:         "site",
	Size:         "size",
	Subbed:       "subbed",
	Title:        "title",
	Unrated:      "unrated",
	Upscaled:     "upscaled",
	Volumes:      "volumes",
	Year:         "year",
}

var Columns = []string{
	Column.Hash,
	Column.TorrentTitle,

	Column.Source,
	Column.Category,
	Column.ImdbId,
	Column.CreatedAt,
	Column.UpdatedAt,
	Column.ParsedAt,
	Column.ParserVersion,
	Column.SentAt,

	Column.Audio,
	Column.BitDepth,
	Column.Channel,
	Column.Codec,
	Column.Complete,
	Column.Container,
	Column.Date,
	Column.Documentary,
	Column.Dubbed,
	Column.Edition,
	Column.Episodes,
	Column.Extension,
	Column.ReleaseGroup,
	Column.HDR,
	Column.Hardcoded,
	Column.Is3D,
	Column.Languages,
	Column.Network,
	Column.Proper,
	Column.Quality,
	Column.Remastered,
	Column.Repack,
	Column.Resolution,
	Column.Retail,
	Column.Seasons,
	Column.Site,
	Column.Size,
	Column.Subbed,
	Column.Title,
	Column.Unrated,
	Column.Upscaled,
	Column.Volumes,
	Column.Year,
}

var get_by_hash_query = fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`,
	`"`+strings.Join(Columns, `","`)+`"`,
	TableName,
	Column.Hash,
)

func GetByHash(hash string) (*TorrentInfo, error) {
	row := db.QueryRow(get_by_hash_query, hash)

	var tInfo TorrentInfo
	if err := row.Scan(
		&tInfo.Hash,
		&tInfo.TorrentTitle,

		&tInfo.Source,
		&tInfo.Category,
		&tInfo.ImdbId,
		&tInfo.CreatedAt,
		&tInfo.UpdatedAt,
		&tInfo.ParsedAt,
		&tInfo.ParserVersion,
		&tInfo.SentAt,

		&tInfo.Audio,
		&tInfo.BitDepth,
		&tInfo.Channel,
		&tInfo.Codec,
		&tInfo.Complete,
		&tInfo.Container,
		&tInfo.Date,
		&tInfo.Documentary,
		&tInfo.Dubbed,
		&tInfo.Edition,
		&tInfo.Episodes,
		&tInfo.Extension,
		&tInfo.ReleaseGroup,
		&tInfo.HDR,
		&tInfo.Hardcoded,
		&tInfo.Is3D,
		&tInfo.Languages,
		&tInfo.Network,
		&tInfo.Proper,
		&tInfo.Quality,
		&tInfo.Remastered,
		&tInfo.Repack,
		&tInfo.Resolution,
		&tInfo.Retail,
		&tInfo.Seasons,
		&tInfo.Site,
		&tInfo.Size,
		&tInfo.Subbed,
		&tInfo.Title,
		&tInfo.Unrated,
		&tInfo.Upscaled,
		&tInfo.Volumes,
		&tInfo.Year,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &tInfo, nil
}

var get_by_hashes_query = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IN ",
	`"`+strings.Join(Columns, `","`)+`"`,
	TableName,
	Column.Hash,
)

func GetByHashes(hashes []string) (map[string]TorrentInfo, error) {
	byHash := map[string]TorrentInfo{}

	if len(hashes) == 0 {
		return byHash, nil
	}

	query := fmt.Sprintf("%s (%s)", get_by_hashes_query, util.RepeatJoin("?", len(hashes), ","))
	args := make([]any, len(hashes))
	for i, hash := range hashes {
		args[i] = hash
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		tInfo := TorrentInfo{}
		if err := rows.Scan(
			&tInfo.Hash,
			&tInfo.TorrentTitle,

			&tInfo.Source,
			&tInfo.Category,
			&tInfo.ImdbId,
			&tInfo.CreatedAt,
			&tInfo.UpdatedAt,
			&tInfo.ParsedAt,
			&tInfo.ParserVersion,
			&tInfo.SentAt,

			&tInfo.Audio,
			&tInfo.BitDepth,
			&tInfo.Channel,
			&tInfo.Codec,
			&tInfo.Complete,
			&tInfo.Container,
			&tInfo.Date,
			&tInfo.Documentary,
			&tInfo.Dubbed,
			&tInfo.Edition,
			&tInfo.Episodes,
			&tInfo.Extension,
			&tInfo.ReleaseGroup,
			&tInfo.HDR,
			&tInfo.Hardcoded,
			&tInfo.Is3D,
			&tInfo.Languages,
			&tInfo.Network,
			&tInfo.Proper,
			&tInfo.Quality,
			&tInfo.Remastered,
			&tInfo.Repack,
			&tInfo.Resolution,
			&tInfo.Retail,
			&tInfo.Seasons,
			&tInfo.Site,
			&tInfo.Size,
			&tInfo.Subbed,
			&tInfo.Title,
			&tInfo.Unrated,
			&tInfo.Upscaled,
			&tInfo.Volumes,
			&tInfo.Year,
		); err != nil {
			return nil, err
		}
		byHash[tInfo.Hash] = tInfo
	}

	return byHash, nil
}

type TorrentInfoInsertDataFile struct {
	Name string
	Idx  int
	Size int64
}

type TorrentInfoInsertData struct {
	Hash         string
	TorrentTitle string
	Size         int64
	Source       TorrentInfoSource
	Category     TorrentInfoCategory

	File TorrentInfoInsertDataFile
}

var insert_query_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TableName,
	strings.Join([]string{
		Column.Hash,
		Column.TorrentTitle,
		Column.Size,
		Column.Source,
		Column.Category,
	}, ","),
)
var insert_query_values_placeholder = "(" + util.RepeatJoin("?", 5, ",") + ")"
var insert_query_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s = CASE WHEN %s.%s = -1 THEN EXCLUDED.%s ELSE %s.%s END`,
	Column.Hash,
	Column.Size,
	TableName,
	Column.Size,
	Column.Size,
	TableName,
	Column.Size,
)

func get_insert_query(count int) string {
	return insert_query_before_values + util.RepeatJoin(insert_query_values_placeholder, count, ",") + insert_query_after_values
}

func Insert(items []TorrentInfoInsertData, sid string) {
	count := len(items)
	if count == 0 || !strings.HasPrefix(sid, "tt") {
		return
	}

	category := TorrentInfoCategoryMovie
	if strings.Contains(sid, ":") {
		category = TorrentInfoCategorySeries
	}

	streamsBySource := map[TorrentInfoSource][]TorrentStreamInsertData{}
	query := get_insert_query(count)
	args := make([]any, 0, 5*count)
	for _, t := range items {
		if _, ok := streamsBySource[t.Source]; !ok {
			streamsBySource[t.Source] = []TorrentStreamInsertData{}
		}
		streamsBySource[t.Source] = append(streamsBySource[t.Source], TorrentStreamInsertData{
			Hash: t.Hash,
			File: t.File,
		})

		args = append(args, t.Hash, t.TorrentTitle, t.Size, t.Source, category)
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		log.Error("failed to insert torrent info", "count", count, "error", err)
	}
	go InsertStreams(streamsBySource, sid)
}
