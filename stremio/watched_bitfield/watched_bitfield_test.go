package stremio_watched_bitfield

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAndModify(t *testing.T) {
	videos := []string{
		"tt2934286:1:1",
		"tt2934286:1:2",
		"tt2934286:1:3",
		"tt2934286:1:4",
		"tt2934286:1:5",
		"tt2934286:1:6",
		"tt2934286:1:7",
		"tt2934286:1:8",
		"tt2934286:1:9",
	}
	watched := "tt2934286:1:5:5:eJyTZwAAAEAAIA=="

	wb, err := NewWatchedBitFieldFromString(watched, videos)
	assert.NoError(t, err)

	assert.True(t, wb.GetVideo("tt2934286:1:5"))
	assert.False(t, wb.GetVideo("tt2934286:1:6"))

	serialized, err := wb.String()
	assert.NoError(t, err)

	rt_wb, err := NewWatchedBitFieldFromString(serialized, videos)
	assert.NoError(t, err)

	assert.True(t, rt_wb.GetVideo("tt2934286:1:5"))
	assert.False(t, rt_wb.GetVideo("tt2934286:1:6"))

	wb.SetVideo("tt2934286:1:6", true)
	assert.True(t, wb.GetVideo("tt2934286:1:6"))
}

func TestConstructFromArray(t *testing.T) {
	arr := make([]bool, 500)
	video_ids := make([]string, 500)
	for i := range 500 {
		video_ids[i] = "tt2934286:1:" + strconv.Itoa(i+1)
	}

	wb := NewWatchedBitFieldFromArray(arr, video_ids)

	// All should be false
	for i, video_id := range video_ids {
		assert.False(t, wb.Get(i))
		assert.False(t, wb.GetVideo(video_id))
	}

	// Set half to true
	for i := range video_ids {
		wb.Set(i, i%2 == 0)
	}

	// Serialize and deserialize to new structure
	serialized, err := wb.String()
	assert.NoError(t, err)
	wb2, err := NewWatchedBitFieldFromString(serialized, video_ids)
	assert.NoError(t, err)

	// Half should still be true
	for i, video_id := range video_ids {
		assert.Equal(t, i%2 == 0, wb2.Get(i))
		assert.Equal(t, i%2 == 0, wb2.GetVideo(video_id))
	}
}

func TestToStringEmpty(t *testing.T) {
	watched := NewWatchedBitFieldFromArray([]bool{}, []string{})
	serialized, err := watched.String()
	assert.NoError(t, err)

	assert.True(t, strings.HasPrefix(serialized, "undefined:1:"))
}

func TestWatchedFieldDeserialize(t *testing.T) {
	str := "tt7767422:3:8:24:eJz7//8/AAX9Av4="
	json_value_bytes, _ := json.Marshal(str)
	json_value := string(json_value_bytes)

	expected := WatchedField{}
	assert.NoError(t, expected.UnmarshalText([]byte(str)), "Should parse field")

	actual_from_json := WatchedField{}
	assert.NoError(t, json.Unmarshal([]byte(json_value), &actual_from_json), "Should deserialize")

	assert.Equal(t, expected, actual_from_json)

	assert.Equal(t, 24, actual_from_json.AnchorLength)
	assert.Equal(t, "tt7767422:3:8", actual_from_json.AnchorVideo)
}

func TestDeserializeEmpty(t *testing.T) {
	watched, err := NewWatchedBitFieldFromString("undefined:1:eJwDAAAAAAE=", []string{})
	assert.NoError(t, err)

	expected := NewWatchedBitField(NewBitField8(0), []string{})
	assert.Equal(t, expected.bitfield.Length, watched.bitfield.Length)
	assert.Equal(t, expected.video_ids, watched.video_ids)
}
