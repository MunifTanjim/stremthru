package util

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5Hash(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}
