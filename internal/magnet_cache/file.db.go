package magnet_cache

import (
	"log"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const FileTableName = "magnet_cache_file"

type MagnetCacheFile struct {
	Hash string `json:"-"`
	Name string `json:"n"`
	Idx  int    `json:"i"`
	Size int    `json:"s"`
	SId  string `json:"-"`
}

func GetFilesByHashes(hashes []string) (map[string]Files, error) {
	args := make([]interface{}, len(hashes))
	hashPlaceholders := make([]string, len(hashes))
	for i, hash := range hashes {
		args[i] = hash
		hashPlaceholders[i] = "?"
	}

	rows, err := db.Query("SELECT h, "+db.FnJSONGroupArray+"("+db.FnJSONObject+"('i', i, 'n', n, 's', s)) AS files FROM "+FileTableName+" WHERE h IN ("+strings.Join(hashPlaceholders, ",")+") GROUP BY h", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byHash := map[string]Files{}
	for rows.Next() {
		hash := ""
		files := Files{}
		if err := rows.Scan(&hash, &files); err != nil {
			return nil, err
		}
		byHash[hash] = files
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return byHash, nil
}

func TrackFiles(hash string, files Files, sid string) {
	if len(files) == 0 {
		return
	}

	var query strings.Builder
	query.WriteString("INSERT INTO " + FileTableName + " (h,i,n,s,sid) VALUES ")
	placeholder := "(?,?,?,?,?)"
	count := 0

	var args []interface{}

	for _, file := range files {
		if count > 0 {
			query.WriteString(",")
		}
		query.WriteString(placeholder)
		fsid := file.SId
		if fsid == "" {
			fsid = sid
		}
		args = append(args, hash, file.Idx, file.Name, file.Size, fsid)
		count++
	}

	query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + FileTableName + ".i = -1 THEN excluded.i ELSE " + FileTableName + ".i END, s = CASE WHEN " + FileTableName + ".s = -1 THEN excluded.s ELSE " + FileTableName + ".s END, sid = excluded.sid")
	_, err := db.Exec(query.String(), args...)
	if err != nil {
		log.Printf("[magnet_cache_file] failed to track: %v\n", err)
	}
}

func BulkTrackFiles(filesByHash map[string]Files, sid string) {
	var query strings.Builder
	query.WriteString("INSERT INTO " + FileTableName + " (h,i,n,s,sid) VALUES ")
	placeholder := "(?,?,?,?,?)"
	count := 0

	var args []interface{}

	for hash, files := range filesByHash {
		for _, file := range files {
			if count > 0 {
				query.WriteString(",")
			}
			query.WriteString(placeholder)
			fsid := file.SId
			if fsid == "" {
				fsid = sid
			}
			args = append(args, hash, file.Idx, file.Name, file.Size, fsid)
			count++
		}
	}

	if count > 0 {
		query.WriteString(" ON CONFLICT (h, n) DO UPDATE SET i = CASE WHEN " + FileTableName + ".i = -1 THEN excluded.i ELSE " + FileTableName + ".i END, s = CASE WHEN " + FileTableName + ".s = -1 THEN excluded.s ELSE " + FileTableName + ".s END, sid = excluded.sid")
		_, err := db.Exec(query.String(), args...)
		if err != nil {
			log.Printf("[magnet_cache_file] failed to bulk track: %v\n", err)
		}
	}
}
