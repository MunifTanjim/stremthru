package db

import (
	"database/sql/driver"
	"errors"
	"time"
)

type Timestamp struct{ time.Time }

func (t Timestamp) IsZero() bool {
	if t.Time.IsZero() {
		return true
	}
	return t.Unix() <= 0
}

func (t Timestamp) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	if Dialect == DBDialectPostgres {
		return t.Time, nil
	}
	return t.Unix(), nil
}

func (t *Timestamp) Scan(value any) error {
	switch v := value.(type) {
	case int64:
		t.Time = time.Unix(v, 0)
	case time.Time:
		t.Time = v
	case nil:
		t.Time = time.Unix(0, 0)
	default:
		return errors.New("failed to convert value to db.Timestamp")
	}
	return nil
}

type DateOnly struct{ time.Time }

func (t DateOnly) IsZero() bool {
	if t.Time.IsZero() {
		return true
	}
	return t.Unix() <= 0
}

func (t DateOnly) String() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Format(time.DateOnly)
}

func (t DateOnly) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(time.DateOnly), nil
}

func (t *DateOnly) Scan(value any) error {
	switch v := value.(type) {
	case string:
		if v == "" {
			t.Time = time.Unix(0, 0)
		} else {
			t.Time, _ = time.Parse(time.DateOnly, v)
		}
	case time.Time:
		t.Time = v
	case nil:
		t.Time = time.Unix(0, 0)
	default:
		return errors.New("failed to convert value to db.Timestamp")
	}
	return nil
}
