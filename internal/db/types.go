package db

import (
	"database/sql/driver"
	"errors"
	"time"
)

type Timestamp struct{ time.Time }

func (t Timestamp) Value() (driver.Value, error) {
	return t.Unix(), nil
}

func (t *Timestamp) Scan(value interface{}) error {
	switch v := value.(type) {
	case int64:
		t.Time = time.Unix(v, 0)
	case time.Time:
		t.Time = v
	default:
		return errors.New("failed to convert value to db.Timestamp")
	}
	return nil
}
