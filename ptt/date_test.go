package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestDate(t *testing.T) {
	for _, tc := range []struct {
		ttitle string
		date   string
	}{
		{"Stephen Colbert 2019 10 25 Eddie Murphy 480p x264-mSD [eztv]", "2019-10-25"},
		{"Stephen Colbert 25/10/2019 Eddie Murphy 480p x264-mSD [eztv]", "2019-10-25"},
		{"Jimmy.Fallon.2020.02.14.Steve.Buscemi.WEB.x264-XLF[TGx]", "2020-02-14"},
		{"The Young And The Restless - S43 E10986 - 2016-08-12", "2016-08-12"},
		{"Indias Best Dramebaaz 2 Ep 19 (13 Feb 2016) HDTV x264-AquoTube", "2016-02-13"},
		{"07 2015 YR/YR 07-06-15.mp4", "2015-07-06"},
		{"SIX.S01E05.400p.229mb.hdtv.x264-][ Collateral ][ 16-Feb-2017 mp4", "2017-02-16"},
		{"SIX.S01E05.400p.229mb.hdtv.x264-][ Collateral ][ 16-Feb-17 mp4", "2017-02-16"},
		{"WWE Smackdown - 11/21/17 - 21st November 2017 - Full Show", "2017-11-21"},
		{"WWE RAW 9th Dec 2019 WEBRip h264-TJ [TJET]", "2019-12-09"},
		{"WWE RAW 1st Dec 2019 WEBRip h264-TJ [TJET]", "2019-12-01"},
		{"WWE RAW 2nd Dec 2019 WEBRip h264-TJ [TJET]", "2019-12-02"},
		{"WWE RAW 3rd Dec 2019 WEBRip h264-TJ [TJET]", "2019-12-03"},
		{"EastEnders_20200116_19302000.mp4", "2020-01-16"},
		{"AEW DARK 4th December 2020 WEBRip h264-TJ", "2020-12-04"},
		{"AEW DARK 4th November 2020 WEBRip h264-TJ", "2020-11-04"},
		{"AEW DARK 4th October 2020 WEBRip h264-TJ", "2020-10-04"},
		{"WWE NXT 30th Sept 2020 WEBRip h264-TJ", "2020-09-30"},
		{"AEW DARK 4th September 2020 WEBRip h264-TJ", "2020-09-04"},
		{"WWE Main Event 6th August 2020 WEBRip h264-TJ", "2020-08-06"},
		{"WWE Main Event 4th July 2020 WEBRip h264-TJ", "2020-07-04"},
		{"WWE Main Event 4th June 2020 WEBRip h264-TJ", "2020-06-04"},
		{"WWE Main Event 4th May 2020 WEBRip h264-TJ", "2020-05-04"},
		{"WWE Main Event 4th April 2020 WEBRip h264-TJ", "2020-04-04"},
		{"WWE Main Event 3rd March 2020 WEBRip h264-TJ", "2020-03-03"},
		{"WWE Main Event 2nd February 2020 WEBRip h264-TJ", "2020-02-02"},
		{"WWE Main Event 1st January 2020 WEBRip h264-TJ", "2020-01-01"},
		{"wwf.raw.is.war.18.09.00.avi", "2000-09-18"},
		{"The Colbert Report - 10-30-2010 - Rally to Restore Sanity and or Fear.avi", "2010-10-30"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.date, result.Date)
		})
	}

	for _, tc := range []struct {
		name   string
		ttitle string
	}{
		{"not detect date from series title", "11 22 63 - Temporada 1 [HDTV][Cap.103][Español Castellano]"},
		{"not detect date from movie title", "September 30 1955 1977 1080p BluRay"},
		{"not detect date from movie title v2", "11-11-11.2011.1080p.BluRay.x264.DTS-FGT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Empty(t, result.Date)
		})
	}
}
