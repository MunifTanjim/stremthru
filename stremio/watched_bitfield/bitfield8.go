package stremio_watched_bitfield

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"slices"
)

type BitField8 struct {
	Length int
	values []byte
}

func (bf *BitField8) Get(i int) bool {
	index := i / 8
	bit := i % 8
	if index >= len(bf.values) {
		return false
	}
	return (bf.values[index]>>bit)&1 != 0
}

func (bf *BitField8) Set(i int, val bool) {
	index := i / 8
	mask := byte(1 << (i % 8))

	if index >= len(bf.values) {
		bf.values = slices.Grow(bf.values, index-len(bf.values)+1)
		bf.Length = len(bf.values) * 8
	}

	if val {
		bf.values[index] |= mask
	} else {
		bf.values[index] &^= mask
	}
}

func (bf *BitField8) FirstIndexOf(val bool) int {
	for i := 0; i < bf.Length; i++ {
		if bf.Get(i) == val {
			return i
		}
	}
	return -1
}

func (bf *BitField8) LastIndexOf(val bool) int {
	for i := bf.Length - 1; i >= 0; i-- {
		if bf.Get(i) == val {
			return i
		}
	}
	return -1
}

func (bf *BitField8) MarshalText() (text []byte, err error) {
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, 6)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(bf.values)
	if err != nil {
		writer.Close()
		return nil, NewError(ErrCodeUnexpected, "failed to write compressed data").WithCause(err)
	}

	err = writer.Close()
	if err != nil {
		return nil, NewError(ErrCodeUnexpected, "failed to finalize compression").WithCause(err)
	}

	return base64.StdEncoding.AppendEncode(text, buf.Bytes()), nil
}

func (bf *BitField8) UnmarshalText(text []byte) error {
	var packed []byte
	packed, err := base64.StdEncoding.AppendDecode(packed, text)
	if err != nil {
		return NewError(ErrCodeInvalidFormat, "failed to decode bitfield").WithCause(err)
	}
	reader, err := zlib.NewReader(bytes.NewReader(packed))
	if err != nil {
		return NewError(ErrCodeUnexpected, "failed to create zlib reader").WithCause(err)
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return NewError(ErrCodeInvalidFormat, "failed to decompress data").WithCause(err)
	}
	*bf = *NewBitField8WithValues(decoded, bf.Length)
	return nil
}

func (bf *BitField8) String() (string, error) {
	text, err := bf.MarshalText()
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func (bf *BitField8) MarshalJSON() ([]byte, error) {
	str, err := bf.String()
	if err != nil {
		return nil, err
	}
	return json.Marshal(str)
}

func (bf *BitField8) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	return bf.UnmarshalText([]byte(str))
}

func NewBitField8(length int) *BitField8 {
	length = int(math.Ceil(float64(length) / 8))
	return &BitField8{
		Length: length,
		values: make([]byte, length),
	}
}

func NewBitField8FromString(encoded string, length int) (*BitField8, error) {
	bf := BitField8{Length: length}
	err := bf.UnmarshalText([]byte(encoded))
	if err != nil {
		return nil, err
	}
	return &bf, nil
}

// Creates a new [`BitField8`] using the passed values and an optional
// length for the struct.
//
// If length is `0` a default value of `values.len() * 8` will be used.
func NewBitField8WithValues(values []byte, length int) *BitField8 {
	if length == 0 {
		length = len(values) * 8
	}
	bytes := int(math.Ceil(float64(length) / 8))
	if bytes > len(values) {
		values = slices.Grow(values, bytes-len(values))
	}
	return &BitField8{
		Length: length,
		values: values,
	}
}
