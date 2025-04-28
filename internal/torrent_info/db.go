package torrent_info

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
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
	TorrentInfoSourceDMM         TorrentInfoSource = "dmm"
	TorrentInfoSourceMediaFusion TorrentInfoSource = "mfn"
	TorrentInfoSourceTorrentio   TorrentInfoSource = "tio"
	TorrentInfoSourceAllDebrid   TorrentInfoSource = "ad"
	TorrentInfoSourceDebridLink  TorrentInfoSource = "dl"
	TorrentInfoSourceEasyDebrid  TorrentInfoSource = "ed"
	TorrentInfoSourceOffcloud    TorrentInfoSource = "oc"
	TorrentInfoSourcePikPak      TorrentInfoSource = "pp"
	TorrentInfoSourcePremiumize  TorrentInfoSource = "pm"
	TorrentInfoSourceRealDebrid  TorrentInfoSource = "rd"
	TorrentInfoSourceTorBox      TorrentInfoSource = "tb"
	TorrentInfoSourceUnknown     TorrentInfoSource = ""
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

	Source        string              `json:"src"`
	Category      TorrentInfoCategory `json:"category"`
	CreatedAt     db.Timestamp        `json:"created_at"`
	UpdatedAt     db.Timestamp        `json:"updated_at"`
	ParsedAt      db.Timestamp        `json:"parsed_at"`
	ParserVersion int                 `json:"parser_version"`
	ParserInput   string              `json:"parser_input"`

	Audio       CommaSeperatedString `json:"audio"`
	BitDepth    string               `json:"bit_depth"`
	Channels    CommaSeperatedString `json:"channels"`
	Codec       string               `json:"codec"`
	Commentary  bool                 `json:"commentary"`
	Complete    bool                 `json:"complete"`
	Container   string               `json:"container"`
	Convert     bool                 `json:"convert"`
	Date        db.DateOnly          `json:"date"`
	Documentary bool                 `json:"documentary"`
	Dubbed      bool                 `json:"dubbed"`
	Edition     string               `json:"edition"`
	EpisodeCode string               `json:"episode_code"`
	Episodes    CommaSeperatedInt    `json:"episodes"`
	Extended    bool                 `json:"extended"`
	Extension   string               `json:"extension"`
	Group       string               `json:"group"`
	HDR         CommaSeperatedString `json:"hdr"`
	Hardcoded   bool                 `json:"hardcoded"`
	Languages   CommaSeperatedString `json:"languages"`
	Network     string               `json:"network"`
	Proper      bool                 `json:"proper"`
	Quality     string               `json:"quality"`
	Region      string               `json:"region"`
	Remastered  bool                 `json:"remastered"`
	Repack      bool                 `json:"repack"`
	Resolution  string               `json:"resolution"`
	Retail      bool                 `json:"retail"`
	Seasons     CommaSeperatedInt    `json:"seasons"`
	Site        string               `json:"site"`
	Size        int64                `json:"size"`
	Subbed      bool                 `json:"subbed"`
	ThreeD      string               `json:"three_d"`
	Title       string               `json:"title"`
	Uncensored  bool                 `json:"uncensored"`
	Unrated     bool                 `json:"unrated"`
	Upscaled    bool                 `json:"upscaled"`
	Volumes     CommaSeperatedInt    `json:"volumes"`
	Year        int                  `json:"year"`
	YearEnd     int                  `json:"year_end"`
}

func (ti TorrentInfo) IsParsed() bool {
	return ti.TorrentTitle == ti.ParserInput
}

const TableName = "torrent_info"

type ColumnStruct struct {
	Hash         string
	TorrentTitle string

	Source        string
	Category      string
	CreatedAt     string
	UpdatedAt     string
	ParsedAt      string
	ParserVersion string
	ParserInput   string

	Audio       string
	BitDepth    string
	Channels    string
	Codec       string
	Commentary  string
	Complete    string
	Container   string
	Convert     string
	Date        string
	Documentary string
	Dubbed      string
	Edition     string
	EpisodeCode string
	Episodes    string
	Extended    string
	Extension   string
	Group       string
	HDR         string
	Hardcoded   string
	Languages   string
	Network     string
	Proper      string
	Quality     string
	Region      string
	Remastered  string
	Repack      string
	Resolution  string
	Retail      string
	Seasons     string
	Site        string
	Size        string
	Subbed      string
	ThreeD      string
	Title       string
	Uncensored  string
	Unrated     string
	Upscaled    string
	Volumes     string
	Year        string
	YearEnd     string
}

var Column = ColumnStruct{
	Hash:         "hash",
	TorrentTitle: "t_title",

	Source:        "src",
	Category:      "category",
	CreatedAt:     "created_at",
	UpdatedAt:     "updated_at",
	ParsedAt:      "parsed_at",
	ParserVersion: "parser_version",
	ParserInput:   "parser_input",

	Audio:       "audio",
	BitDepth:    "bit_depth",
	Channels:    "channels",
	Codec:       "codec",
	Commentary:  "commentary",
	Complete:    "complete",
	Container:   "container",
	Convert:     "convert",
	Date:        "date",
	Documentary: "documentary",
	Dubbed:      "dubbed",
	Edition:     "edition",
	EpisodeCode: "episode_code",
	Episodes:    "episodes",
	Extended:    "extended",
	Extension:   "extension",
	Group:       "group",
	HDR:         "hdr",
	Hardcoded:   "hardcoded",
	Languages:   "languages",
	Network:     "network",
	Proper:      "proper",
	Quality:     "quality",
	Region:      "region",
	Remastered:  "remastered",
	Repack:      "repack",
	Resolution:  "resolution",
	Retail:      "retail",
	Seasons:     "seasons",
	Site:        "site",
	Size:        "size",
	Subbed:      "subbed",
	ThreeD:      "three_d",
	Title:       "title",
	Uncensored:  "uncensored",
	Unrated:     "unrated",
	Upscaled:    "upscaled",
	Volumes:     "volumes",
	Year:        "year",
	YearEnd:     "year_end",
}

var Columns = []string{
	Column.Hash,
	Column.TorrentTitle,

	Column.Source,
	Column.Category,
	Column.CreatedAt,
	Column.UpdatedAt,
	Column.ParsedAt,
	Column.ParserVersion,
	Column.ParserInput,

	Column.Audio,
	Column.BitDepth,
	Column.Channels,
	Column.Codec,
	Column.Commentary,
	Column.Complete,
	Column.Container,
	Column.Convert,
	Column.Date,
	Column.Documentary,
	Column.Dubbed,
	Column.Edition,
	Column.EpisodeCode,
	Column.Episodes,
	Column.Extended,
	Column.Extension,
	Column.Group,
	Column.HDR,
	Column.Hardcoded,
	Column.Languages,
	Column.Network,
	Column.Proper,
	Column.Quality,
	Column.Region,
	Column.Remastered,
	Column.Repack,
	Column.Resolution,
	Column.Retail,
	Column.Seasons,
	Column.Site,
	Column.Size,
	Column.Subbed,
	Column.ThreeD,
	Column.Title,
	Column.Uncensored,
	Column.Unrated,
	Column.Upscaled,
	Column.Volumes,
	Column.Year,
	Column.YearEnd,
}

var get_by_hash_query = fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(Columns...),
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
		&tInfo.CreatedAt,
		&tInfo.UpdatedAt,
		&tInfo.ParsedAt,
		&tInfo.ParserVersion,
		&tInfo.ParserInput,

		&tInfo.Audio,
		&tInfo.BitDepth,
		&tInfo.Channels,
		&tInfo.Codec,
		&tInfo.Commentary,
		&tInfo.Complete,
		&tInfo.Container,
		&tInfo.Convert,
		&tInfo.Date,
		&tInfo.Documentary,
		&tInfo.Dubbed,
		&tInfo.Edition,
		&tInfo.EpisodeCode,
		&tInfo.Episodes,
		&tInfo.Extended,
		&tInfo.Extension,
		&tInfo.Group,
		&tInfo.HDR,
		&tInfo.Hardcoded,
		&tInfo.Languages,
		&tInfo.Network,
		&tInfo.Proper,
		&tInfo.Quality,
		&tInfo.Region,
		&tInfo.Remastered,
		&tInfo.Repack,
		&tInfo.Resolution,
		&tInfo.Retail,
		&tInfo.Seasons,
		&tInfo.Site,
		&tInfo.Size,
		&tInfo.Subbed,
		&tInfo.ThreeD,
		&tInfo.Title,
		&tInfo.Uncensored,
		&tInfo.Unrated,
		&tInfo.Upscaled,
		&tInfo.Volumes,
		&tInfo.Year,
		&tInfo.YearEnd,
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
			&tInfo.CreatedAt,
			&tInfo.UpdatedAt,
			&tInfo.ParsedAt,
			&tInfo.ParserVersion,
			&tInfo.ParserInput,

			&tInfo.Audio,
			&tInfo.BitDepth,
			&tInfo.Channels,
			&tInfo.Codec,
			&tInfo.Commentary,
			&tInfo.Complete,
			&tInfo.Container,
			&tInfo.Convert,
			&tInfo.Date,
			&tInfo.Documentary,
			&tInfo.Dubbed,
			&tInfo.Edition,
			&tInfo.EpisodeCode,
			&tInfo.Episodes,
			&tInfo.Extended,
			&tInfo.Extension,
			&tInfo.Group,
			&tInfo.HDR,
			&tInfo.Hardcoded,
			&tInfo.Languages,
			&tInfo.Network,
			&tInfo.Proper,
			&tInfo.Quality,
			&tInfo.Region,
			&tInfo.Remastered,
			&tInfo.Repack,
			&tInfo.Resolution,
			&tInfo.Retail,
			&tInfo.Seasons,
			&tInfo.Site,
			&tInfo.Size,
			&tInfo.Subbed,
			&tInfo.ThreeD,
			&tInfo.Title,
			&tInfo.Uncensored,
			&tInfo.Unrated,
			&tInfo.Upscaled,
			&tInfo.Volumes,
			&tInfo.Year,
			&tInfo.YearEnd,
		); err != nil {
			return nil, err
		}
		byHash[tInfo.Hash] = tInfo
	}

	return byHash, nil
}

type TorrentInfoInsertDataFile = ts.File

type TorrentInfoInsertData = TorrentItem

var insert_query_before_values = fmt.Sprintf(
	`INSERT INTO %s AS ti (%s) VALUES `,
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
var insert_query_on_conflict = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s, %s, %s, %s, %s`,
	Column.Hash,
	fmt.Sprintf(
		"%s = CASE WHEN ti.%s NOT IN ('tio','ad','dl','rd') THEN EXCLUDED.%s ELSE ti.%s END",
		Column.TorrentTitle,
		Column.Source,
		Column.TorrentTitle,
		Column.TorrentTitle,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ti.%s = -1 THEN EXCLUDED.%s ELSE ti.%s END",
		Column.Size,
		Column.Size,
		Column.Size,
		Column.Size,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ti.%s NOT IN ('tio','ad','dl','rd') THEN EXCLUDED.%s ELSE ti.%s END",
		Column.Source,
		Column.Source,
		Column.Source,
		Column.Source,
	),
	fmt.Sprintf(
		"%s = CASE WHEN ti.%s = '' THEN EXCLUDED.%s ELSE ti.%s END",
		Column.Category,
		Column.Category,
		Column.Category,
		Column.Category,
	),
	fmt.Sprintf(
		"%s = ",
		Column.UpdatedAt,
	),
)

func get_insert_query(count int) string {
	return insert_query_before_values +
		util.RepeatJoin(insert_query_values_placeholder, count, ",") +
		insert_query_on_conflict + db.CurrentTimestamp
}

func Upsert(items []TorrentInfoInsertData, category TorrentInfoCategory, discardFileIdx bool) {
	if len(items) == 0 {
		return
	}

	streamItems := []ts.InsertData{}

	for cItems := range slices.Chunk(items, 200) {
		count := len(cItems)
		seenHash := map[string]struct{}{}
		args := make([]any, 0, 5*count)
		for _, t := range cItems {
			if _, seen := seenHash[t.Hash]; seen {
				count--
				continue
			}
			seenHash[t.Hash] = struct{}{}

			tSource := string(t.Source)
			for _, f := range t.Files {
				if f.Source == "" {
					f.Source = tSource
				}
				streamItems = append(streamItems, ts.InsertData{
					Hash: t.Hash,
					File: f,
				})
			}

			if t.TorrentTitle == "" || t.TorrentTitle == t.Hash || strings.HasPrefix(t.TorrentTitle, "magnet:?") {
				count--
				continue
			}

			tCategory := t.Category
			if tCategory == "" {
				tCategory = category
			}

			args = append(args, t.Hash, t.TorrentTitle, t.Size, t.Source, tCategory)
		}

		if count == 0 {
			continue
		}

		query := get_insert_query(count)
		_, err := db.Exec(query, args...)
		if err != nil {
			log.Error("failed to upsert torrent info", "count", count, "error", err)
		} else {
			log.Debug("upserted torrent info", "count", count)
		}
	}

	ts.Record(streamItems, discardFileIdx)
}

var get_unparsed_query = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s != %s LIMIT ?",
	db.JoinColumnNames(Columns...),
	TableName,
	Column.TorrentTitle,
	Column.ParserInput,
)

func GetUnparsed(limit int) ([]TorrentInfo, error) {
	if limit == 0 {
		limit = 5000
	}

	rows, err := db.Query(get_unparsed_query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tInfos := []TorrentInfo{}
	for rows.Next() {
		tInfo := TorrentInfo{}
		if err := rows.Scan(
			&tInfo.Hash,
			&tInfo.TorrentTitle,

			&tInfo.Source,
			&tInfo.Category,
			&tInfo.CreatedAt,
			&tInfo.UpdatedAt,
			&tInfo.ParsedAt,
			&tInfo.ParserVersion,
			&tInfo.ParserInput,

			&tInfo.Audio,
			&tInfo.BitDepth,
			&tInfo.Channels,
			&tInfo.Codec,
			&tInfo.Commentary,
			&tInfo.Complete,
			&tInfo.Container,
			&tInfo.Convert,
			&tInfo.Date,
			&tInfo.Documentary,
			&tInfo.Dubbed,
			&tInfo.Edition,
			&tInfo.EpisodeCode,
			&tInfo.Episodes,
			&tInfo.Extended,
			&tInfo.Extension,
			&tInfo.Group,
			&tInfo.HDR,
			&tInfo.Hardcoded,
			&tInfo.Languages,
			&tInfo.Network,
			&tInfo.Proper,
			&tInfo.Quality,
			&tInfo.Region,
			&tInfo.Remastered,
			&tInfo.Repack,
			&tInfo.Resolution,
			&tInfo.Retail,
			&tInfo.Seasons,
			&tInfo.Site,
			&tInfo.Size,
			&tInfo.Subbed,
			&tInfo.ThreeD,
			&tInfo.Title,
			&tInfo.Uncensored,
			&tInfo.Unrated,
			&tInfo.Upscaled,
			&tInfo.Volumes,
			&tInfo.Year,
			&tInfo.YearEnd,
		); err != nil {
			return nil, err
		}
		tInfos = append(tInfos, tInfo)
	}

	return tInfos, nil
}

var upsert_parsed_on_conflict_columns = append([]string{
	Column.ParserVersion,
	Column.ParserInput,
}, Columns[slices.Index(Columns, Column.Audio):]...)
var upsert_parsed_query_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TableName,
	db.JoinColumnNames(Columns...),
)
var upsert_parsed_query_values_placeholder = fmt.Sprintf("(%s)", util.RepeatJoin("?", len(Columns), ","))
var upsert_parsed_query_on_confict = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET (%s) = (%s), (%s, %s) = `,
	Column.Hash,
	db.JoinColumnNames(upsert_parsed_on_conflict_columns...),
	strings.Join(
		func() []string {
			cols := make([]string, len(upsert_parsed_on_conflict_columns))
			for i := range upsert_parsed_on_conflict_columns {
				cols[i] = `EXCLUDED."` + upsert_parsed_on_conflict_columns[i] + `"`
			}
			return cols
		}(),
		",",
	),
	Column.ParsedAt,
	Column.UpdatedAt,
)

func get_upsert_parsed_query(count int) string {
	return upsert_parsed_query_before_values +
		util.RepeatJoin(upsert_parsed_query_values_placeholder, count, ",") +
		upsert_parsed_query_on_confict +
		"(" + db.CurrentTimestamp + "," + db.CurrentTimestamp + ")"
}

func UpsertParsed(tInfos []*TorrentInfo) error {
	for cTInfos := range slices.Chunk(tInfos, 200) {
		count := len(cTInfos)
		query := get_upsert_parsed_query(count)

		args := make([]any, 0, len(Columns)*count)
		for i := range cTInfos {
			tInfo := cTInfos[i]
			args = append(
				args,

				tInfo.Hash,
				tInfo.TorrentTitle,

				tInfo.Source,
				tInfo.Category,
				tInfo.CreatedAt,
				tInfo.UpdatedAt,
				tInfo.ParsedAt,
				tInfo.ParserVersion,
				tInfo.ParserInput,

				tInfo.Audio,
				tInfo.BitDepth,
				tInfo.Channels,
				tInfo.Codec,
				tInfo.Commentary,
				tInfo.Complete,
				tInfo.Container,
				tInfo.Convert,
				tInfo.Date,
				tInfo.Documentary,
				tInfo.Dubbed,
				tInfo.Edition,
				tInfo.EpisodeCode,
				tInfo.Episodes,
				tInfo.Extended,
				tInfo.Extension,
				tInfo.Group,
				tInfo.HDR,
				tInfo.Hardcoded,
				tInfo.Languages,
				tInfo.Network,
				tInfo.Proper,
				tInfo.Quality,
				tInfo.Region,
				tInfo.Remastered,
				tInfo.Repack,
				tInfo.Resolution,
				tInfo.Retail,
				tInfo.Seasons,
				tInfo.Site,
				tInfo.Size,
				tInfo.Subbed,
				tInfo.ThreeD,
				tInfo.Title,
				tInfo.Uncensored,
				tInfo.Unrated,
				tInfo.Upscaled,
				tInfo.Volumes,
				tInfo.Year,
				tInfo.YearEnd,
			)
		}

		if _, err := db.Exec(query, args...); err != nil {
			return err
		}
	}
	return nil
}

type TorrentItem struct {
	Hash         string              `json:"hash"`
	TorrentTitle string              `json:"name"`
	Size         int64               `json:"size"`
	Source       TorrentInfoSource   `json:"src"`
	Category     TorrentInfoCategory `json:"category"`

	Files ts.Files `json:"files"`
}

type ListTorrentsData struct {
	Items      []TorrentItem `json:"items"`
	TotalItems int           `json:"total_items"`
}

var list_query_columns = strings.Join(
	func() []string {
		columns := []string{Column.Hash, Column.TorrentTitle, Column.Size, Column.Source, Column.Category}
		cols := make([]string, 5)
		for i := range columns {
			cols[i] = `ti."` + columns[i] + `"`
		}
		return cols
	}(),
	",",
)

func get_list_query_select() string {
	return fmt.Sprintf(
		"SELECT %s, %s(%s('n',ts.%s,'i',ts.%s,'s',ts.%s,'sid',ts.%s,'src',ts.%s)) AS files",
		list_query_columns,
		db.FnJSONGroupArray,
		db.FnJSONObject,
		ts.Column.Name,
		ts.Column.Idx,
		ts.Column.Size,
		ts.Column.SId,
		ts.Column.Source,
	)
}

var list_query_after_select = fmt.Sprintf(
	" FROM %s ti LEFT JOIN %s ts ON ti.%s = ts.%s AND ts.%s != ''",
	TableName,
	ts.TableName,
	Column.Hash,
	ts.Column.Hash,
	ts.Column.Source,
)
var list_query_cond_hashes = fmt.Sprintf(
	"%s IN (SELECT DISTINCT %s FROM %s WHERE %s = ? OR %s LIKE ?)",
	Column.Hash,
	ts.Column.Hash,
	ts.TableName,
	ts.Column.SId,
	ts.Column.SId,
)
var list_query_cond_no_missing_size = fmt.Sprintf(
	"%s > 0",
	Column.Size,
)
var list_query_after_cond = fmt.Sprintf(
	" GROUP BY %s",
	Column.Hash,
)

func get_list_query(excludeMissingSize bool) string {
	query := get_list_query_select() + list_query_after_select + " WHERE " + list_query_cond_hashes
	if excludeMissingSize {
		query += " AND " + list_query_cond_no_missing_size
	}
	query += list_query_after_cond
	return query
}

func ListByStremId(stremId string, excludeMissingSize bool) (*ListTorrentsData, error) {
	args := make([]any, 2)
	if strings.Contains(stremId, ":") {
		args[0] = stremId
		args[1], _, _ = strings.Cut(stremId, ":")
	} else {
		args[0] = stremId
		args[1] = stremId + ":%"
	}

	query := get_list_query(excludeMissingSize)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Error("failed to list torrents by strem id", "error", err, "stremId", stremId)
		return nil, err
	}
	defer rows.Close()

	items := []TorrentItem{}
	for rows.Next() {
		var item TorrentItem
		if err := rows.Scan(&item.Hash, &item.TorrentTitle, &item.Size, &item.Source, &item.Category, &item.Files); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	data := &ListTorrentsData{
		Items:      items,
		TotalItems: len(items),
	}
	return data, nil
}

var dump_torrents_query = fmt.Sprintf(`
SELECT ti.%s,
       ti.%s,
       CASE WHEN ti.%s > 0 THEN ti.%s ELSE COALESCE(SUM(ts.%s), -1) END,
       (ti.%s <= 0)
FROM %s ti
         LEFT JOIN %s ts
                   ON ti.%s <= 0 AND ts.%s = ti.%s AND ts.%s >= 0
                       AND ts.%s != '' AND ts.%s NOT LIKE '%%:%%'
GROUP BY ti.%s
`,
	Column.Hash,
	Column.TorrentTitle,
	Column.Size, Column.Size, ts.Column.Size,
	Column.Size,
	TableName,
	ts.TableName,
	Column.Size, ts.Column.Hash, Column.Hash, ts.Column.Size,
	ts.Column.SId, ts.Column.SId,
	Column.Hash,
)

type DumpTorrentsItem struct {
	Hash         string `json:"hash"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	IsSizeApprox bool   `json:"_size_approx"`
}

func DumpTorrents(noApproxSize bool, noMissingSize bool) ([]DumpTorrentsItem, error) {
	rows, err := db.Query(dump_torrents_query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []DumpTorrentsItem{}
	for rows.Next() {
		var item DumpTorrentsItem
		if err := rows.Scan(&item.Hash, &item.Name, &item.Size, &item.IsSizeApprox); err != nil {
			return nil, err
		}
		if noApproxSize && item.IsSizeApprox {
			item.Size = -1
			item.IsSizeApprox = false
		}
		if noMissingSize && item.Size <= 0 {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

type Stats struct {
	TotalCount    int            `json:"total_count"`
	CountBySource map[string]int `json:"count_by_source"`
	Streams       *ts.Stats      `json:"streams,omitempty"`
}

var stats_query = fmt.Sprintf(
	"SELECT %s, COUNT(%s) FROM %s GROUP BY %s",
	Column.Source,
	Column.Hash,
	TableName,
	Column.Source,
)

func GetStats() (*Stats, error) {
	totalCount := 0
	rows, err := db.Query(stats_query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	countBySource := map[string]int{}
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		countBySource[source] = count
		totalCount += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stats := &Stats{
		CountBySource: countBySource,
		TotalCount:    totalCount,
	}
	if tsStats, err := ts.GetStats(); err != nil {
		log.Error("failed to get torrent stream stats", "error", err)
	} else {
		stats.Streams = tsStats
	}
	return stats, nil
}

var exists_by_hash_query = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s IN ",
	Column.Hash,
	TableName,
	Column.Hash,
)

func ExistsByHash(hashes []string) (map[string]bool, error) {
	exists := make(map[string]bool, len(hashes))
	if len(hashes) == 0 {
		return exists, nil
	}

	for cHashes := range slices.Chunk(hashes, 2000) {
		query := exists_by_hash_query + "(" + util.RepeatJoin("?", len(cHashes), ",") + ")"
		args := make([]any, len(cHashes))
		for i, hash := range cHashes {
			args[i] = hash
		}
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var hash string
			if err := rows.Scan(&hash); err != nil {
				return nil, err
			}
			exists[hash] = true
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return exists, nil
}

var query_get_unmapped_hashes = fmt.Sprintf(
	"SELECT ti.%s FROM %s ti LEFT JOIN %s ito ON ti.%s = ito.%s WHERE ito.%s IS NULL LIMIT ?",
	Column.Hash,
	TableName,
	imdb_torrent.TableName,
	Column.Hash,
	imdb_torrent.Column.Hash,
	imdb_torrent.Column.TId,
)

func GetUnmappedHashes(limit int) ([]string, error) {
	hashes := []string{}
	limit = max(1, min(limit, 20000))

	rows, err := db.Query(query_get_unmapped_hashes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hashes, nil
}
