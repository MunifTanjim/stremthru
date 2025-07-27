package core

import "encoding/base64"

func Base64Encode(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func Base64EncodeToByte(value string) (encoded []byte) {
	return base64.StdEncoding.AppendEncode(encoded, []byte(value))
}

func Base64EncodeByte(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

func Base64Decode(value string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	return string(decodedBytes), err
}

func Base64DecodeToByte(value string) ([]byte, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	return decodedBytes, err
}
