package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RedirectConfigTestSuite struct {
	suite.Suite
}

func (s *RedirectConfigTestSuite) TestParseRedirectConfig() {
	rcm := parseRedirectConfig("torbox-signup:https://a,,https://b")
	s.Len(rcm, 1)
	s.Equal([]string{"https://a", "https://b"}, rcm["torbox-signup"])

	rcm = parseRedirectConfig("torbox-signup:https://a,https://b realdebrid-signup:http://x")
	s.Len(rcm, 2)
	s.Equal([]string{"https://a", "https://b"}, rcm["torbox-signup"])
	s.Equal([]string{"http://x"}, rcm["realdebrid-signup"])

	rcm = parseRedirectConfig("")
	s.Len(rcm, 0)
}

func (s *RedirectConfigTestSuite) TestGetRandom() {
	rcm := parseRedirectConfig("key-a:https://a key-b:https://b1,https://b2")

	s.Equal("", rcm.GetRandom("key-missing"))

	s.Equal("https://a", rcm.GetRandom("key-a"))

	seen := make(map[string]bool)
	for range 128 {
		url := rcm.GetRandom("key-b")
		s.Contains([]string{"https://b1", "https://b2"}, url)
		seen[url] = true
		if seen["https://b1"] && seen["https://b2"] {
			break
		}
	}
	s.True(seen["https://b1"], "expected https://b1 to be returned at least once")
	s.True(seen["https://b2"], "expected https://b2 to be returned at least once")
}

func TestRedirectConfig(t *testing.T) {
	suite.Run(t, new(RedirectConfigTestSuite))
}
