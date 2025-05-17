package stremio_wrap

import (
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var ExtractRequestBaseURL = shared.ExtractRequestBaseURL

var SendResponse = stremio_shared.SendResponse
var SendHTML = stremio_shared.SendHTML

func dedupeStreams(allStreams []WrappedStream) []WrappedStream {
	hashSeen := map[string]struct{}{}

	streams := []WrappedStream{}
	for i := range allStreams {
		s := allStreams[i]
		if s.r != nil && s.r.Hash != "" {
			if _, seen := hashSeen[s.r.Hash]; seen {
				continue
			} else {
				hashSeen[s.r.Hash] = struct{}{}
			}
		}
		streams = append(streams, s)
	}
	return streams
}
