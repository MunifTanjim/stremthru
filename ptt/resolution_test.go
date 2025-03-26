package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolution(t *testing.T) {
	for _, test := range []struct {
		name       string
		ttitle     string
		resolution string
	}{
		{"1080P", "Annabelle.2014.1080p.PROPER.HC.WEBRip.x264.AAC.2.0-RARBG", "1080p"},
		{"720P", "doctor_who_2005.8x12.death_in_heaven.720p_hdtv_x264-fov", "720p"},
		{"720P with uppercase", "UFC 187 PPV 720P HDTV X264-KYR", "720p"},
		{"4K", "The Smurfs 2 2013 COMPLETE FULL BLURAY UHD (4K) - IPT EXCLUSIVE", "4k"},
		{"2060P as 4K", "Joker.2019.2160p.4K.BluRay.x265.10bit.HDR.AAC5.1", "4k"},
		{"Custom Aspect Ratio for 4K", "[Beatrice-Raws] Evangelion 3.333 You Can (Not) Redo [BDRip 3840x1632 HEVC TrueHD]", "4k"},
		{"Custom Aspect Ratio for 1080P", "[Erai-raws] Evangelion 3.0 You Can (Not) Redo - Movie [1920x960][Multiple Subtitle].mkv", "1080p"},
		{"Custom Aspect Ratio for 720P", "[JacobSwaggedUp] Kizumonogatari I: Tekketsu-hen (BD 1280x544) [MP4 Movie]", "720p"},
		{"720i as 720P", "UFC 187 PPV 720i HDTV X264-KYR", "720p"},
		{"Typo in 720P", "IT Chapter Two.2019.7200p.AMZN WEB-DL.H264.[Eng Hin Tam Tel]DDP 5.1.MSubs.D0T.Telly", "720p"},
		{"Typo in 1080P", "Dumbo (1941) BRRip XvidHD 10800p-NPW", "1080p"},
		{"1080 without spaces and M prefix", "BluesBrothers2M1080.www.newpct.com.mkv", "1080p"},
		{"1080 without spaces and BD prefix", "BenHurParte2BD1080.www.newpct.com.mkv", "1080p"},
		{"720 without spaces title prefix", "1993720p_101_WWW.NEWPCT1.COM.mkvv", "720p"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := Parse(test.ttitle)
			assert.Equal(t, test.resolution, r.Resolution)
		})
	}

	// PY
	for _, test := range []struct {
		ttitle     string
		resolution string
	}{
		{"The Boys S04E01 E02 E03 4k to 1080p AMZN WEBrip x265 DDP5 1 D0c", "1080p"},
		{"Batman Returns 1992 4K Remastered BluRay 1080p DTS AC3 x264-MgB", "1080p"},
		{"Life After People (2008) [1080P.BLURAY] [720p] [BluRay] [YTS.MX]", "720p"},
	} {
		t.Run(test.ttitle, func(t *testing.T) {
			r := Parse(test.ttitle)
			assert.Equal(t, test.resolution, r.Resolution)
		})
	}
}
