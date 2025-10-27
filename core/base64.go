package core

import (
	"bytes"
	"encoding/base64"
	"io"
)

func Base64Encode(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func Base64EncodeToByte(value string) (encoded []byte) {
	return base64.StdEncoding.AppendEncode(encoded, []byte(value))
}

func Base64EncodeByte(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

func Base64EncodeFile(value io.Reader) (string, error) {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(encoder, value); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Base64Decode(value string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	return string(decodedBytes), err
}

func Base64DecodeToByte(value string) ([]byte, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	return decodedBytes, err
}
