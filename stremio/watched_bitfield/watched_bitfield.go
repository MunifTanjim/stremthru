package stremio_watched_bitfield

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

// (De)Serializable field that tracks which videos have been watched
// and the latest one watched.
//
// This is a [`WatchedBitField`] compatible field, (de)serialized
// without the knowledge of `videos_ids`.
//
// `{anchor:video_id}:{anchor_length}:{bitfield8}`
type WatchedField struct {
	// The anchor video id
	//
	// Indicates which is the last watched video id.
	AnchorVideo string
	// The length from the beginning of the `BitField8` to the last
	// watched video.
	AnchorLength int
	BitField     *BitField8
}

func (wf *WatchedField) MarshalText() (text []byte, err error) {
	bitfield_str, err := wf.BitField.String()
	if err != nil {
		return nil, err
	}
	return []byte(wf.AnchorVideo + ":" + strconv.Itoa(wf.AnchorLength) + ":" + bitfield_str), nil
}

func (wf *WatchedField) UnmarshalText(text []byte) error {
	// serialized is formed by {id}:{len}:{serializedBuf}, but since {id} might contain : we have to pop gradually and then keep the rest
	components := strings.Split(string(text), ":")
	if len(components) < 3 {
		return NewError(ErrCodeInvalidFormat, "Not enough components")
	}

	bitfield_buf := components[len(components)-1]

	anchor_length, err := strconv.Atoi(components[len(components)-2])
	if err != nil {
		return NewError(ErrCodeInvalidFormat, "Cannot obtain the length field").WithCause(err)
	}

	anchor_video_id := strings.Join(components[:len(components)-2], ":")

	wf.BitField = &BitField8{}
	err = wf.BitField.UnmarshalText([]byte(bitfield_buf))
	if err != nil {
		return err
	}

	wf.AnchorVideo = anchor_video_id
	wf.AnchorLength = anchor_length
	return nil
}

func (wf *WatchedField) String() (string, error) {
	text, err := wf.MarshalText()
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func (wf *WatchedField) MarshalJSON() ([]byte, error) {
	str, err := wf.String()
	if err != nil {
		return nil, err
	}
	return json.Marshal(str)
}

func (wf *WatchedField) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	return wf.UnmarshalText([]byte(str))
}

func NewWatchedFieldFromWatchedBitField(wb *WatchedBitField) *WatchedField {
	last_id := max(wb.bitfield.LastIndexOf(true), 0)

	last_video_id := "undefined"
	if last_id < len(wb.video_ids) {
		last_video_id = wb.video_ids[last_id]
	}

	return &WatchedField{
		AnchorVideo:  last_video_id,
		AnchorLength: last_id + 1,
		BitField:     wb.bitfield,
	}
}

func (wf *WatchedField) toWatchedBitField(video_ids []string) (*WatchedBitField, error) {
	anchor_video_idx := slices.Index(video_ids, wf.AnchorVideo)
	if anchor_video_idx != -1 {
		offset := wf.AnchorLength - anchor_video_idx - 1
		bitfield := NewBitField8WithValues(wf.BitField.values, len(video_ids))

		// in case of a previous empty array, this will be 0
		if offset != 0 {
			// Resize the buffer
			resized_wbf := &WatchedBitField{
				bitfield:  NewBitField8(len(video_ids)),
				video_ids: video_ids,
			}

			// rewrite the old buf into the new one, applying the offset
			for i := range video_ids {
				id_in_prev := i + offset
				if id_in_prev >= 0 && id_in_prev < bitfield.Length {
					resized_wbf.Set(i, bitfield.Get(id_in_prev))
				}
			}
			return resized_wbf, nil
		}

		return &WatchedBitField{
			bitfield:  bitfield,
			video_ids: video_ids,
		}, nil
	}

	// videoId could not be found, return a totally blank buf
	return &WatchedBitField{
		bitfield:  NewBitField8(len(video_ids)),
		video_ids: video_ids,
	}, nil

}

// Tracks which videos have been watched.
//
// Serialized in the format `{id}:{len}:{serializedBuf}` but since `{id}`
// might contain `:` we pop gradually and then keep the rest.
type WatchedBitField struct {
	bitfield  *BitField8
	video_ids []string
}

func (wbf *WatchedBitField) MarshalText() (text []byte, err error) {
	wf := NewWatchedFieldFromWatchedBitField(wbf)
	return wf.MarshalText()
}

func (wbf *WatchedBitField) UnmarshalText(text []byte) error {
	wf := WatchedField{}
	err := wf.UnmarshalText(text)
	if err != nil {
		return err
	}
	wbf_converted, err := wf.toWatchedBitField(wbf.video_ids)
	if err != nil {
		return err
	}
	wbf.bitfield = wbf_converted.bitfield
	return nil
}

func (wbf *WatchedBitField) String() (string, error) {
	text, err := wbf.MarshalText()
	if err != nil {
		return "", err
	}
	return string(text), nil
}

func (wbf *WatchedBitField) Get(idx int) bool {
	return wbf.bitfield.Get(idx)
}

func (wbf *WatchedBitField) GetVideo(video_id string) bool {
	pos := slices.Index(wbf.video_ids, video_id)
	if pos == -1 {
		return false
	}
	return wbf.bitfield.Get(pos)
}

func (wbf *WatchedBitField) Set(idx int, v bool) {
	wbf.bitfield.Set(idx, v)
}

func (wbf *WatchedBitField) SetVideo(video_id string, v bool) {
	pos := slices.Index(wbf.video_ids, video_id)
	if pos == -1 {
		return
	}
	wbf.bitfield.Set(pos, v)
}

func NewWatchedBitField(bitfield *BitField8, video_ids []string) *WatchedBitField {
	return &WatchedBitField{
		bitfield:  bitfield,
		video_ids: video_ids,
	}
}

func NewWatchedBitFieldFromString(str string, video_ids []string) (*WatchedBitField, error) {
	wbf := WatchedBitField{video_ids: video_ids}
	err := wbf.UnmarshalText([]byte(str))
	return &wbf, err
}

func NewWatchedBitFieldFromArray(arr []bool, video_ids []string) *WatchedBitField {
	bitfield := NewBitField8(len(video_ids))
	for i, v := range arr {
		bitfield.Set(i, v)
	}
	return NewWatchedBitField(bitfield, video_ids)
}
