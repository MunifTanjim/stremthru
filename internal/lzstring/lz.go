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

// map of "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-$"
var keyStrUriSafe map[byte]int = map[byte]int{74: 9, 78: 13, 83: 18, 36: 64, 109: 38, 114: 43, 116: 45, 101: 30, 45: 63, 73: 8, 81: 16, 113: 42, 49: 53, 50: 54, 54: 58, 76: 11, 100: 29, 107: 36, 121: 50, 77: 12, 89: 24, 105: 34, 66: 1, 69: 4, 85: 20, 48: 52, 119: 48, 117: 46, 120: 49, 52: 56, 56: 60, 110: 39, 112: 41, 70: 5, 71: 6, 79: 14, 88: 23, 97: 26, 102: 31, 103: 32, 67: 2, 118: 47, 65: 0, 68: 3, 72: 7, 108: 37, 51: 55, 57: 61, 82: 17, 90: 25, 98: 27, 115: 44, 122: 51, 53: 57, 86: 21, 106: 35, 111: 40, 55: 59, 43: 62, 75: 10, 80: 15, 84: 19, 87: 22, 99: 28, 104: 33}

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
	return keyStrUriSafe[char]
}

// Input is composed of ASCII characters, so accessing it by array has no UTF-8 pb.
func readBits(nb int, data *dataStruct) int {
	result := 0
	power := 1
	for i := 0; i < nb; i++ {
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
