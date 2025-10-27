package bitmagnet

import (
	"database/sql"
	"time"

	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
)

type Torrent struct {
	Hash        string
	Title       string
	Seeders     int
	Leechers    int
	Private     bool
	Size        int64
	FilesCount  int
	FilesStatus string
	Files       ts.Files
	ContentType string
	UpdatedAt   time.Time
}

var query_get_torrents = `
SELECT encode(tc.info_hash, 'hex'::text)  AS hash,
       min(t.name)                        AS t_title,
       min(coalesce(tc.seeders, 0))       AS seeders,
       min(coalesce(tc.leechers, 0))      AS leechers,
       bool_or(t.private)                 AS private,
       min(tc.size)                       AS size,
       min(tc.files_count)                AS files_count,
       min(t.files_status)                AS files_status,
       json_agg(json_build_object('i', coalesce(tf.index, 0),
                                  'p', coalesce(tf.path, t.name),
                                  's', coalesce(tf.size, tc.size))
                ORDER BY tf.index)        AS files,
       coalesce(min(tc.content_type), '') AS content_type,
       min(tc.updated_at)                 AS updated_at
FROM torrent_contents tc
         LEFT JOIN torrents t ON tc.info_hash = t.info_hash
         LEFT JOIN torrent_files tf on t.info_hash = tf.info_hash
WHERE tc.id IN (SELECT tc.id
                       FROM torrent_contents tc
                       WHERE tc.updated_at >= $1
                         AND tc.content_type IN ('movie', 'tv_show')
                       ORDER BY tc.updated_at
                       LIMIT $2 OFFSET $3)
GROUP BY tc.id
ORDER BY min(tc.updated_at), tc.id
`

func GetTorrents(db *sql.DB, limit, offset int, cursor_updated_at time.Time) ([]Torrent, error) {
	rows, err := db.Query(query_get_torrents, cursor_updated_at, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	torrents := make([]Torrent, 0, limit)
	for rows.Next() {
		t := Torrent{}
		if err := rows.Scan(
			&t.Hash,
			&t.Title,
			&t.Seeders,
			&t.Leechers,
			&t.Private,
			&t.Size,
			&t.FilesCount,
			&t.FilesStatus,
			&t.Files,
			&t.ContentType,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		torrents = append(torrents, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return torrents, nil
}
