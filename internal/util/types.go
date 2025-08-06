package util

import "encoding/json"

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
