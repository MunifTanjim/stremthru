package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParseNewzQueryHeaderTestSuite struct {
	suite.Suite
}

func (s *ParseNewzQueryHeaderTestSuite) parse(queryBlob string) newzIndexerRequestHeaderMap {
	return parseNewzIndexerRequestHeader(queryBlob, "")
}

func (s *ParseNewzQueryHeaderTestSuite) TestMultipleHeadersWithoutQueryType() {
	result := s.parse("User-Agent: Mozilla/5.0\nAccept: text/html")
	s.Equal("Mozilla/5.0", result.Query["*"].Get("User-Agent"))
	s.Equal("text/html", result.Query["*"].Get("Accept"))
}

func (s *ParseNewzQueryHeaderTestSuite) TestSingleQueryTypeWithPreset() {
	result := s.parse("[tv]\n:sonarr:")
	s.Contains(result.Query["tv"].Get("User-Agent"), "Sonarr")
}

func (s *ParseNewzQueryHeaderTestSuite) TestMultipleQueryTypes() {
	result := s.parse("[tv]\nUser-Agent: Sonarr\n\n[movie]\nUser-Agent: Radarr")
	s.Equal("Sonarr", result.Query["tv"].Get("User-Agent"))
	s.Equal("Radarr", result.Query["movie"].Get("User-Agent"))
}

func (s *ParseNewzQueryHeaderTestSuite) TestMixedPresetsAndCustomHeaders() {
	result := s.parse("[tv]\n:sonarr:\nCustom-Header: custom-value")
	s.Contains(result.Query["tv"].Get("User-Agent"), "Sonarr")
	s.Equal("custom-value", result.Query["tv"].Get("Custom-Header"))
}

func (s *ParseNewzQueryHeaderTestSuite) TestInvalidPresetPanics() {
	s.Panics(func() {
		s.parse(":nonexistent:")
	})
}

func (s *ParseNewzQueryHeaderTestSuite) TestMissingQueryTypeAfterEmptyLinePanics() {
	s.Panics(func() {
		s.parse("[tv]\nUser-Agent: Sonarr\n\nAccept: text/html")
	})
}

func TestParseNewzQueryHeader(t *testing.T) {
	suite.Run(t, new(ParseNewzQueryHeaderTestSuite))
}

type ParseNewzGrabHeaderTestSuite struct {
	suite.Suite
}

func (s *ParseNewzGrabHeaderTestSuite) parse(grabBlob string) newzIndexerRequestHeaderMap {
	return parseNewzIndexerRequestHeader("", grabBlob)
}

func (s *ParseNewzGrabHeaderTestSuite) TestMultipleHeaders() {
	result := s.parse("User-Agent: Mozilla/5.0\nAccept: text/html")
	s.Equal("Mozilla/5.0", result.Grab.Get("User-Agent"))
	s.Equal("text/html", result.Grab.Get("Accept"))
}

func (s *ParseNewzGrabHeaderTestSuite) TestPreset() {
	result := s.parse(":chrome:")
	s.NotEmpty(result.Grab.Get("User-Agent"))
}

func (s *ParseNewzGrabHeaderTestSuite) TestQueryTypePanics() {
	s.Panics(func() {
		s.parse("[tv]\nUser-Agent: Sonarr")
	})
}

func TestParseNewzGrabHeader(t *testing.T) {
	suite.Run(t, new(ParseNewzGrabHeaderTestSuite))
}
