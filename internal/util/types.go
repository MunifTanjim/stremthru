package util

import (
	"encoding/json"
	"time"
)

type JSONNumber json.Number

func (rn *JSONNumber) UnmarshalJSON(data []byte) error {
	if string(data) == `""` || string(data) == `null` {
		*rn = "0"
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*rn = JSONNumber(n)
	return nil
}

type JSONTime struct {
	time.Time
	raw []byte
}

func (rt *JSONTime) UnmarshalJSON(data []byte) error {
	if string(data) == `""` || string(data) == `null` {
		*rt = JSONTime{Time: time.Time{}, raw: data}
		return nil
	}
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	*rt = JSONTime{Time: t, raw: data}
	return nil
}

func (rt JSONTime) MarshalJSON() ([]byte, error) {
	if rt.IsZero() {
		return rt.raw, nil
	}
	return json.Marshal(rt.Time)
}
