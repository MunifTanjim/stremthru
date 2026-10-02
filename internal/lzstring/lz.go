package lzstring

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

//
// Decompress uri encoded lz-string
// http://pieroxy.net/blog/pages/lz-string/index.html
// https://github.com/pieroxy/lz-string/
//

const keyStrUriSafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-$"

var baseValueByChar = func() (m [256]int) {
	for i := range len(keyStrUriSafe) {
		m[keyStrUriSafe[i]] = i
	}
	return m
}()

type dataStruct struct {
	input      string
	val        int
	position   int
	index      int
	dictionary []string
	enlargeIn  float64
	numBits    int
}

func getBaseValue(char byte) int {
	return baseValueByChar[char]
}

// Input is composed of ASCII characters, so accessing it by array has no UTF-8 pb.
func readBits(nb int, data *dataStruct) int {
	result := 0
	power := 1
	for range nb {
		respB := data.val & data.position
		data.position = data.position / 2
		if data.position == 0 {
			data.position = 32
			data.val = getBaseValue(data.input[data.index])
			data.index += 1
		}
		if respB > 0 {
			result |= power
		}
		power *= 2
	}
	return result
}

func appendValue(data *dataStruct, str string) {
	data.dictionary = append(data.dictionary, str)
	data.enlargeIn -= 1
	if data.enlargeIn == 0 {
		data.enlargeIn = math.Pow(2, float64(data.numBits))
		data.numBits += 1
	}
}

func getString(last string, data *dataStruct) (string, bool, error) {
	c := readBits(data.numBits, data)
	switch c {
	case 0:
		str := string(rune(readBits(8, data)))
		appendValue(data, str)
		return str, false, nil
	case 1:
		str := string(rune(readBits(16, data)))
		appendValue(data, str)
		return str, false, nil
	case 2:
		return "", true, nil
	}
	if c < len(data.dictionary) {
		return data.dictionary[c], false, nil
	}
	if c == len(data.dictionary) {
		return concatWithFirstRune(last, last), false, nil
	}
	return "", false, errors.New("Bad character encoding.")
}

// Need to handle UTF-8, so we need to use rune to concatenate
func concatWithFirstRune(str string, getFirstRune string) string {
	r, _ := utf8.DecodeRuneInString(getFirstRune)
	return str + string(r)
}

func DecompressFromEncodedUriComponent(input string) (string, error) {
	if input == "" {
		return "", errors.New("input empty")
	}

	data := dataStruct{input, getBaseValue(input[0]), 32, 1, []string{"0", "1", "2"}, 5, 2}

	var r strings.Builder
	result, isEnd, err := getString("", &data)
	r.WriteString(result)
	if err != nil || isEnd {
		return r.String(), err
	}
	last := result
	data.numBits += 1
	for {
		str, isEnd, err := getString(last, &data)
		if err != nil || isEnd {
			return r.String(), err
		}

		r.WriteString(str)
		appendValue(&data, concatWithFirstRune(last, str))
		last = str
	}

	return "", errors.New("Unexpected end of buffer reached.")
}
