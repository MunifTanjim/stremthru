package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceMapIntToString(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  []int
		output []string
	}{
		{"empty slice", []int{}, []string{}},
		{"single element", []int{42}, []string{"42"}},
		{"mixed values", []int{-5, 0, 5}, []string{"-5", "0", "5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.output, SliceMapIntToString(tc.input))
		})
	}
}

func TestSliceMapIntToString_CustomType(t *testing.T) {
	type customInt int
	input := []customInt{1, 2, 3}
	expected := []string{"1", "2", "3"}
	assert.Equal(t, expected, SliceMapIntToString(input))
}
