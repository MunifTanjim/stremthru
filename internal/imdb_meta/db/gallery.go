package db

import (
	"database/sql"
	"time"
)

// GalleryState records that a title's image gallery was fully synced. The
// gallery itself is the title's title_image rows, in pos order.
type GalleryState struct {
	UpdatedAt time.Time
}

// GetGallery returns the sync state, or ErrNotFound if the gallery was never
// fully synced.
func (db *DB) GetGallery(id string) (*GalleryState, error) {
	var uat int64
	err := db.QueryRow(`SELECT uat FROM title_gallery WHERE title_id = ?`, id).Scan(&uat)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &GalleryState{UpdatedAt: time.Unix(uat, 0)}, nil
}

// ListGalleryImages returns up to limit of the title's images starting at
// offset, and the title's total image count.
// Both reads run in one transaction, so the count matches the page.
func (db *DB) ListGalleryImages(id string, limit, offset int) ([]Image, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	var total int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM title_image WHERE title_id = ?`, id).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := tx.Query(
		`SELECT i.id, ti.ep_id, i.kind, i.url, i.width, i.height, i.caption
		 FROM title_image ti JOIN image i ON i.id = ti.image_id
		 WHERE ti.title_id = ? ORDER BY ti.pos LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	images := []Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.EpID, &img.Kind, &img.URL, &img.Width, &img.Height, &img.Caption); err != nil {
			return nil, 0, err
		}
		images = append(images, img)
	}
	return images, total, rows.Err()
}

// AddGalleryImages adds a page of images to the title's gallery after its
// existing members. Blank images are skipped, and images already in the
// gallery keep their place (their data is refreshed), so a sync can be
// repeated or resumed without duplicating anything.
func (db *DB) AddGalleryImages(id string, images []Image) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, img := range images {
		if err := addTitleImage(tx, id, img.EpID, img); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PruneGallery removes the title's gallery members that are not in keep and
// are not the current primary image of the title or one of its episodes, then
// deletes the removed images no title uses any more. Call it only after a
// sync that read every page, with the ids that sync returned.
func (db *DB) PruneGallery(id string, keep []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	kept := make(map[string]bool, len(keep))
	for _, imageID := range keep {
		kept[imageID] = true
	}
	primaries, err := queryStrings(tx,
		`SELECT image_id FROM title WHERE id = ?
		 UNION SELECT t.image_id FROM title_episode te JOIN title t ON t.id = te.id WHERE te.title_id = ?`, id, id)
	if err != nil {
		return err
	}
	for _, imageID := range primaries {
		kept[imageID] = true
	}
	members, err := queryStrings(tx, `SELECT image_id FROM title_image WHERE title_id = ?`, id)
	if err != nil {
		return err
	}
	for _, imageID := range members {
		if kept[imageID] {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM title_image WHERE title_id = ? AND image_id = ?`, id, imageID); err != nil {
			return err
		}
		if err := deleteUnusedImage(tx, imageID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// deleteUnusedImage deletes the image row if no gallery and no title uses it.
func deleteUnusedImage(tx *sql.Tx, imageID string) error {
	_, err := tx.Exec(
		`DELETE FROM image WHERE id = ?
		 AND NOT EXISTS (SELECT 1 FROM title_image WHERE image_id = ?)
		 AND NOT EXISTS (SELECT 1 FROM title WHERE image_id = ?)`, imageID, imageID, imageID)
	return err
}

func queryStrings(tx *sql.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CompleteGallery records that every gallery page was stored. If the title
// was found missing meanwhile (deleted while the sync ran), it removes what
// the sync stored instead and returns ErrNotFound.
func (db *DB) CompleteGallery(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var missing int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM missing WHERE id = ? AND uat >= unixepoch() - ?`, id, int64(missingKept.Seconds()),
	).Scan(&missing); err != nil {
		return err
	}
	if missing > 0 {
		images, err := queryStrings(tx, `SELECT image_id FROM title_image WHERE title_id = ?`, id)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM title_image WHERE title_id = ?`, id); err != nil {
			return err
		}
		for _, imageID := range images {
			if err := deleteUnusedImage(tx, imageID); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrNotFound
	}
	if _, err := tx.Exec(
		`INSERT INTO title_gallery (title_id, uat) VALUES (?, unixepoch())
		 ON CONFLICT (title_id) DO UPDATE SET uat = unixepoch()`, id); err != nil {
		return err
	}
	return tx.Commit()
}
