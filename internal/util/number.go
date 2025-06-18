package util

import "strconv"

func IntRange(start int, end int) []int {
	length := end - start + 1
	if length < 0 {
		return nil
	}
	result := make([]int, length)
	for i := start; i <= end; i++ {
		result[i-start] = i
	}
	return result
}

func SafeParseInt(str string, fallbackValue int) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		return fallbackValue
	}
	return val
}
