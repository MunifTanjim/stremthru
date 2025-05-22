package imdb_title

import (
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type Genre = string

const (
	GenreAction      Genre = "Action"
	GenreAdult       Genre = "Adult"
	GenreAdventure   Genre = "Adventure"
	GenreAnimation   Genre = "Animation"
	GenreBiography   Genre = "Biography"
	GenreComedy      Genre = "Comedy"
	GenreCrime       Genre = "Crime"
	GenreDocumentary Genre = "Documentary"
	GenreDrama       Genre = "Drama"
	GenreFamily      Genre = "Family"
	GenreFantasy     Genre = "Fantasy"
	GenreFilmNoir    Genre = "Film Noir"
	GenreGameShow    Genre = "Game Show"
	GenreHistory     Genre = "History"
	GenreHorror      Genre = "Horror"
	GenreMusical     Genre = "Musical"
	GenreMusic       Genre = "Music"
	GenreMystery     Genre = "Mystery"
	GenreNews        Genre = "News"
	GenreRealityTV   Genre = "Reality-TV"
	GenreRomance     Genre = "Romance"
	GenreSciFi       Genre = "Sci-Fi"
	GenreShort       Genre = "Short"
	GenreSport       Genre = "Sport"
	GenreTalkShow    Genre = "Talk-Show"
	GenreThriller    Genre = "Thriller"
	GenreWar         Genre = "War"
	GenreWestern     Genre = "Western"
)

const GenreTableName = "imdb_title_genre"

type IMDBTitleGenre struct {
	TId   string `json:"tid"`
	Genre Genre  `json:"genre"`
}

type GenreColumnStruct struct {
	TId   string
	Genre Genre
}

var GenreColumn = GenreColumnStruct{
	TId:   "tid",
	Genre: "genre",
}

var query_record_genre_cleanup = fmt.Sprintf(
	`DELETE FROM %s WHERE %s IN `,
	GenreTableName,
	GenreColumn.TId,
)
var query_record_genre_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s) VALUES `,
	GenreTableName,
	GenreColumn.TId,
	GenreColumn.Genre,
)
var query_record_genre_value_placeholder = fmt.Sprintf(
	`(?, ?)`,
)
var query_record_genre_after_values = fmt.Sprintf(
	` ON CONFLICT DO NOTHING`,
)

func recordGenre(tx *db.Tx, metas []IMDBTitleMeta) error {
	count := len(metas)
	if count == 0 {
		return nil
	}

	cleanupArgs := make([]any, 0, count)

	args := make([]any, 0, count*2)

	for i := range metas {
		meta := &metas[i]
		if len(meta.Genres) > 0 {
			cleanupArgs = append(cleanupArgs, meta.TId)
		}
		for _, genre := range meta.Genres {
			args = append(args, meta.TId, genre)
		}
	}

	cleanupQuery := query_record_genre_cleanup + "(" + util.RepeatJoin("?", len(cleanupArgs), ",") + ")"

	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	query := query_record_genre_before_values +
		util.RepeatJoin(query_record_genre_value_placeholder, len(args)/2, ",") +
		query_record_genre_after_values

	_, err := tx.Exec(query, args...)

	return err
}
