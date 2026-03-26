package util

import (
	"fmt"
	"strconv"
)

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

func SafeParseFloat(str string, fallbackValue float64) float64 {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fallbackValue
	}
	return val
}

func ZeroPadInt(n int, length int) string {
	return fmt.Sprintf("%0"+strconv.Itoa(length)+"d", n)
}

func IntToString[T ~int | int8 | int16 | int32 | int64](i T) string {
	return strconv.FormatInt(int64(i), 10)
}
