package util

import "strings"

func PtrToBool(ptr *bool, fallback bool) bool {
	if ptr != nil {
		return *ptr
	}
	return fallback
}

func StringToBool(str string, fallback bool) bool {
	if str == "" {
		return fallback
	}
	switch strings.ToLower(str) {
	case "1", "t", "true", "y", "yes":
		return true
	case "0", "f", "false", "n", "no":
		return false
	default:
		return fallback
	}
}
