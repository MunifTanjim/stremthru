package util

import "strconv"

func SliceMapIntToString(elems []int) []string {
	strElems := make([]string, len(elems))
	for i := range elems {
		strElems[i] = strconv.Itoa(elems[i])
	}
	return strElems
}

func FilterSlice[T any](s []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(s))
	for _, v := range s {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}
