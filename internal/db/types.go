package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
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

type NullString struct {
	String string
}

func (nv NullString) Value() (driver.Value, error) {
	if nv.String == "" {
		return nil, nil
	}
	return nv.String, nil
}

func (nv *NullString) Scan(value any) error {
	switch v := value.(type) {
	case string:
		nv.String = v
	case nil:
		nv.String = ""
	default:
		return errors.New("failed to convert value")
	}
	return nil
}

type CommaSeperatedString []string

func (css CommaSeperatedString) Value() (driver.Value, error) {
	if len(css) == 0 {
		return "", nil
	}
	return "," + strings.Join(css, ",") + ",", nil
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
	*css = strings.Split(strings.Trim(str, ","), ",")
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

type JSONStringList []string

func (list JSONStringList) Value() (driver.Value, error) {
	return json.Marshal(list)
}

func (list *JSONStringList) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, list)
}
