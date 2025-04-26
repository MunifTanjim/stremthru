package util

import "strconv"

func TSVGetValue[T any](row []string, idx int, defaultValue T, nilValue string) (T, error) {
	if len(row) <= idx {
		return defaultValue, nil
	}

	val := row[idx]

	if val == "" || val == nilValue {
		return defaultValue, nil
	}

	switch any(defaultValue).(type) {
	case int:
		v, err := strconv.Atoi(val)
		return T(any(v).(T)), err
	case bool:
		v, err := strconv.ParseBool(val)
		return T(any(v).(T)), err
	default:
		return T(any(val).(T)), nil
	}
}
