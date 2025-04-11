package util

import (
	"math/rand"
	"strings"
)

type charSet struct {
	AlphaNumeric          string
	AlphaNumericMixedCase string
}

var CharSet = charSet{
	AlphaNumeric:          "abcdefghijklmnopqrstuvwxyz0123456789",
	AlphaNumericMixedCase: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
}

func GenerateRandomString(length int, charSet string) string {
	n := len(charSet)
	var sb strings.Builder
	sb.Grow(length)
	for range length {
		sb.WriteByte(charSet[rand.Intn(n)])
	}
	return sb.String()
}
