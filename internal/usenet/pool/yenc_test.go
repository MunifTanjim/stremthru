package usenet_pool

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/mnightingale/rapidyenc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestBytes(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

func encodeYenc(data []byte, filename string, partNum, totalParts int, totalSize, begin int64) []byte {
	var buf bytes.Buffer

	meta := rapidyenc.Meta{
		FileName:   filename,
		FileSize:   totalSize,
		PartNumber: int64(partNum),
		TotalParts: int64(totalParts),
		Offset:     begin - 1,
		PartSize:   int64(len(data)),
	}

	encoder, err := rapidyenc.NewEncoder(&buf, meta)
	if err != nil {
		panic(fmt.Sprintf("failed to create yEnc encoder: %v", err))
	}

	if _, err = encoder.Write(data); err != nil {
		panic(fmt.Sprintf("failed to encode yEnc data: %v", err))
	}

	if err := encoder.Close(); err != nil {
		panic(fmt.Sprintf("failed to close yEnc encoder: %v", err))
	}

	return buf.Bytes()
}

func TestYencDecoder(t *testing.T) {
	t.Run("DecodeSinglePart", func(t *testing.T) {
		originalData := []byte("Hello, World! This is test data for yEnc encoding.")
		encoded := encodeYenc(originalData, "test.txt", 1, 1, int64(len(originalData)), 1)

		decoder := NewYEncDecoder(bytes.NewReader(encoded))
		header, err := decoder.Header()
		require.NoError(t, err)

		assert.Equal(t, "test.txt", header.FileName)
		assert.Equal(t, int64(len(originalData)), header.FileSize)

		decoded, err := io.ReadAll(decoder)
		require.NoError(t, err)
		assert.Equal(t, originalData, decoded)
	})

	t.Run("DecodeMultiPart", func(t *testing.T) {
		totalSize := int64(100)
		part1Data := makeTestBytes(50)
		part2Data := makeTestBytes(50)

		// Part 1: bytes 1-50 (1-based)
		encoded1 := encodeYenc(part1Data, "test.bin", 1, 2, totalSize, 1)
		decoder1 := NewYEncDecoder(bytes.NewReader(encoded1))
		header1, err := decoder1.Header()
		require.NoError(t, err)

		assert.Equal(t, int64(1), header1.PartNumber)
		assert.Equal(t, int64(2), header1.TotalParts)
		assert.Equal(t, int64(1), header1.Begin())
		assert.Equal(t, int64(50), header1.End())

		br := header1.ByteRange()
		assert.Equal(t, int64(0), br.Start)
		assert.Equal(t, int64(50), br.End)

		decoded1, err := io.ReadAll(decoder1)
		require.NoError(t, err)
		assert.Equal(t, part1Data, decoded1)

		// Part 2: bytes 51-100 (1-based)
		encoded2 := encodeYenc(part2Data, "test.bin", 2, 2, totalSize, 51)
		decoder2 := NewYEncDecoder(bytes.NewReader(encoded2))
		header2, err := decoder2.Header()
		require.NoError(t, err)

		assert.Equal(t, int64(2), header2.PartNumber)
		assert.Equal(t, int64(2), header2.TotalParts)
		assert.Equal(t, int64(51), header2.Begin())
		assert.Equal(t, int64(100), header2.End())
	})

	t.Run("ByteRangeConversion", func(t *testing.T) {
		header := &YEncHeader{}
		header.Offset = 0
		header.PartSize = 100

		br := header.ByteRange()
		assert.Equal(t, int64(0), br.Start, "Begin 1 should convert to Start 0")
		assert.Equal(t, int64(100), br.End, "End 100 should stay 100 (exclusive)")
		assert.Equal(t, int64(100), br.Count())
	})

	t.Run("HandleEOF", func(t *testing.T) {
		data := []byte("short")
		encoded := encodeYenc(data, "short.txt", 1, 1, int64(len(data)), 1)

		decoder := NewYEncDecoder(bytes.NewReader(encoded))
		_, err := decoder.Header()
		require.NoError(t, err)

		// Read beyond EOF
		buf := make([]byte, 100)
		n, err := decoder.Read(buf)
		assert.Equal(t, len(data), n)

		// Next read should return EOF
		n, err = decoder.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})
}
