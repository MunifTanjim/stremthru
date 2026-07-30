package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContentCachedStaleTimeTestSuite struct {
	suite.Suite
}

func (s *StoreContentCachedStaleTimeTestSuite) TestStoreContentCachedStaleTime() {
	_, err := parseStoreContentCachedStaleTime("*:12h:4h", true)
	s.ErrorContains(err, "must be at least 18h")

	_, err = parseStoreContentCachedStaleTime("*:18h:4h", true)
	s.ErrorContains(err, "must be at least 6h")

	_, err = parseStoreContentCachedStaleTime("*:1d:8h", true)
	s.ErrorContains(err, "invalid")

	staleTime, err := parseStoreContentCachedStaleTime("*:36h:12h", true)
	s.Nil(err)
	s.Equal(staleTime.GetStaleTime(true, "realdebrid"), 36*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "realdebrid"), 12*time.Hour)
	s.Equal(staleTime.GetStaleTime(true, "torbox"), 36*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "torbox"), 12*time.Hour)

	staleTime, err = parseStoreContentCachedStaleTime("*:36h:12h,realdebrid:48h:16h", true)
	s.Nil(err)
	s.Equal(staleTime.GetStaleTime(true, "realdebrid"), 48*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "realdebrid"), 16*time.Hour)
	s.Equal(staleTime.GetStaleTime(true, "torbox"), 36*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "torbox"), 12*time.Hour)

	staleTime, err = parseStoreContentCachedStaleTime("realdebrid:48h:16h", true)
	s.Nil(err)
	s.Equal(staleTime.GetStaleTime(true, "realdebrid"), 48*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "realdebrid"), 16*time.Hour)
	s.Equal(staleTime.GetStaleTime(true, "torbox"), 24*time.Hour)
	s.Equal(staleTime.GetStaleTime(false, "torbox"), 8*time.Hour)
}

func (s *StoreContentCachedStaleTimeTestSuite) TestStoreContentCachedStaleTimeWithoutPeer() {
	_, err := parseStoreContentCachedStaleTime("*:29m:5m", false)
	s.ErrorContains(err, "must be at least 30m")

	_, err = parseStoreContentCachedStaleTime("*:30m:4m", false)
	s.ErrorContains(err, "must be at least 5m")

	staleTime, err := parseStoreContentCachedStaleTime("*:30m:5m", false)
	s.Nil(err)
	s.Equal(staleTime.GetStaleTime(true, "realdebrid"), 30*time.Minute)
	s.Equal(staleTime.GetStaleTime(false, "realdebrid"), 5*time.Minute)
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(StoreContentCachedStaleTimeTestSuite))
}
