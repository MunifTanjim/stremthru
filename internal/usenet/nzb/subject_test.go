package nzb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubjectParser(t *testing.T) {
	for _, tc := range []struct {
		fileCount int
		subject   string
		number    int
		name      string
	}{
		{
			52,
			`[1/52] - "618ee7f37097e26dbb27464aac1243dc" yEnc  524288000 (1/732)`,
			1,
			"618ee7f37097e26dbb27464aac1243dc",
		},
		{
			0,
			`[N3wZ] \V7JRL9192688\::bdb28593ae0331ab0d1f5c039390b676 yEnc (1/147)`,
			0,
			"bdb28593ae0331ab0d1f5c039390b676",
		},
		{
			16,
			`[PRiVATE]-[WtFnZb]-[Goblin.S01E01.1080p.WEBRip.AAC2.0.H.264-CasStudio.mkv]-[1/16] - "" yEnc  2713638372 (1/3786)`,
			1,
			"Goblin.S01E01.1080p.WEBRip.AAC2.0.H.264-CasStudio.mkv",
		},
		{
			0,
			`[PRiVATE]-[WtFnZb]-[20]-[1/Jane.the.Virgin.S03E01.Chapter.Forty-Five.1080p.BluRay.REMUX.AVC.DD.5.1-EPSiLON.mkv] - "" yEnc (1/[PRiVATE] \ec574c923a\::47c52cea0dd7ae.a8d35a7aae9bf10b1958409b0b6fce.1f60ab3c::/133a18c96893/) 1 (1/0) (1/0)`,
			0,
			"Jane.the.Virgin.S03E01.Chapter.Forty-Five.1080p.BluRay.REMUX.AVC.DD.5.1-EPSiLON.mkv",
		},
		{
			2,
			`[PRiVATE]-[WtFnZb]-[newz[NZB].nfo]-[2/2] - "" yEnc  13 (1/1)`,
			2,
			"newz[NZB].nfo",
		},
		{
			2,
			`[N3wZ] \2J2XBV192688\::[PRiVATE]-[WtFnZb]-[Love.Death.and.Robots.S02E04.Snow.in.the.Desert.1080p.NF.WEB-DL.DD+5.1.Atmos.HDR.HEVC-L0L.mkv]-[1/2] - "" yEnc  670556996 (1/1310)`,
			1,
			"Love.Death.and.Robots.S02E04.Snow.in.the.Desert.1080p.NF.WEB-DL.DD+5.1.Atmos.HDR.HEVC-L0L.mkv",
		},
		{
			0,
			`MythBusters.S01E08.Buried.Alive.720p.HEVC.x265.mkv - 076 of 194 - All 15 seasons being seeded, I'll get and post what I can ]]var quality[[ yEnc(1/1150)`,
			0,
			"MythBusters.S01E08.Buried.Alive.720p.HEVC.x265.mkv",
		},
		{
			1,
			`[N3wZ] \jUESJH192688\::[PRiVATE]-[WtFnZb]-[/NCIS.Hawaii.S02E15.1080p.PMTP.WEBRip.DDP5.1.x264-WhiteHat[rarbg]/NCIS.Hawaii.S02E15.1080p.PMTP.WEB-DL.DDP5.1.x264-WhiteHat.mkv]-[1/1] - "" yEnc  1427339353 (1/1992)`,
			1,
			"NCIS.Hawaii.S02E15.1080p.PMTP.WEB-DL.DDP5.1.x264-WhiteHat.mkv",
		},
		{
			50,
			`[ TrollHD ] - [ 02/50 ] - "NOVA S43E02 Arctic Ghost Ship 1080i HDTV DD2.0 MPEG2-TrollHD.part01.rar" yEnc(1/164)`,
			2,
			"NOVA S43E02 Arctic Ghost Ship 1080i HDTV DD2.0 MPEG2-TrollHD.part01.rar",
		},
		{
			70,
			`[N3wZ] \8ljUOB192688\::184ac48f60732588dee1103638972754 yEnc [6/70] "a38e6ecd1c884fca9483a9e2bf5107a0.part05.rar" yEnc (1/68)`,
			6,
			"a38e6ecd1c884fca9483a9e2bf5107a0.part05.rar",
		},
		{
			2,
			`[PRiVATE]-[WtFnZb]-[/Star.Trek.Picard.S03E02.Part.Two.Disengage.REPACK.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb[eztv.re].mkv-WtF[nZb]/Star.Trek.Picard.S03E02.Part.Two.Disengage.REPACK.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb[eztv.re].mkv]-[1/2] - "" yEnc  2798062438 (1/3904)`,
			1,
			"Star.Trek.Picard.S03E02.Part.Two.Disengage.REPACK.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb[eztv.re].mkv",
		},
		{
			8,
			`[PRiVATE]-[WtFnZb]-[The.Diplomat.S01E01.The.Cinderella.Thing.1080p.NF.WEB-DL.DDP5.1.Atmos.H.264-playWEB.mkv]-[1/8] - "" yEnc  2101113560 (1/2932)`,
			1,
			"The.Diplomat.S01E01.The.Cinderella.Thing.1080p.NF.WEB-DL.DDP5.1.Atmos.H.264-playWEB.mkv",
		},
		{
			0,
			`[ Younger.S04E07.1080p.WEB-DL.DD2.0.H.264-KiNGS ] - "Younger.S04E07.Fever.Pitch.1080p.AMZN.WEB-DL.DDP2.0.H.264-KiNGS.part31.rar" yEnc (01/66)`,
			0,
			"Younger.S04E07.Fever.Pitch.1080p.AMZN.WEB-DL.DDP2.0.H.264-KiNGS.part31.rar",
		},
		{
			0,
			`Great stuff (001/143) - "Filename.txt" yEnc (1/1)`,
			0,
			"Filename.txt",
		},
		{
			0,
			`"910a284f98ebf57f6a531cd96da48838.vol01-03.par2" yEnc (1/3)`,
			0,
			"910a284f98ebf57f6a531cd96da48838.vol01-03.par2",
		},
		{
			30,
			`Subject-KrzpfTest [02/30] - ""KrzpfTest.part.nzb"" yEnc`,
			2,
			"KrzpfTest.part.nzb",
		},
		{
			12,
			`[PRiVATE]-[WtFnZb]-[Supertje-_S03E11-12_-blabla_+_blabla_WEBDL-480p.mkv]-[4/12] - "" yEnc 9786 (1/1366)`,
			4,
			"Supertje-_S03E11-12_-blabla_+_blabla_WEBDL-480p.mkv",
		},
		{
			2,
			`[N3wZ] MAlXD245333\\::[PRiVATE]-[WtFnZb]-[Show.S04E04.720p.AMZN.WEBRip.x264-GalaxyTV.mkv]-[1/2] - "" yEnc  293197257 (1/573)`,
			1,
			"Show.S04E04.720p.AMZN.WEBRip.x264-GalaxyTV.mkv",
		},
		{
			6,
			`reftestnzb bf1664007a71 [1/6] - "20b9152c-57eb-4d02-9586-66e30b8e3ac2" yEnc (1/22) 15728640`,
			1,
			"20b9152c-57eb-4d02-9586-66e30b8e3ac2",
		},
		{
			0,
			"Re: REQ Author Child's The Book-Thanks much - Child, Lee - Author - The Book.epub (1/1)",
			0,
			"REQ Author Child's The Book-Thanks much - Child, Lee - Author - The Book.epub",
		},
		{
			101,
			`63258-0[001/101] - "63258-2.0" yEnc (1/250) (1/250)`,
			1,
			"63258-2.0",
		},
		{
			101,
			`63258-0[001/101] - "63258-2.0toolong" yEnc (1/250) (1/250)`,
			1,
			"63258-2.0toolong",
		},
		{
			25,
			"Singer - A Album (2005) - [04/25] - 02 Sweetest Somebody (I Know).flac",
			4,
			"02 Sweetest Somebody (I Know).flac",
		},
		{
			0,
			"<>random!>",
			0,
			"<>random!>",
		},
		{
			0,
			"nZb]-[Supertje-_S03E11-12_",
			0,
			"nZb]-[Supertje-_S03E11-12_",
		},
		{
			0,
			"Bla [Now it's done.exe]",
			0,
			"Now it's done.exe",
		},
		{
			0,
			"Bla [Now it's done.123nonsense]",
			0,
			"Bla [Now it's done.123nonsense]",
		},
		{
			0,
			`[PRiVATE]-[WtFnZb]-[00000.clpi]-[1/46] - "" yEnc  788 (1/1)`,
			0,
			"00000.clpi",
		},
		{
			0,
			`[PRiVATE]-[WtFnZb]-[Video_(2001)_AC5.1_-RELEASE_[TAoE].mkv]-[1/23] - "" yEnc 1234567890 (1/23456)`,
			0,
			"Video_(2001)_AC5.1_-RELEASE_[TAoE].mkv",
		},
		{
			0,
			"[PRiVATE]-[WtFnZb]-[219]-[1/series.name.s01e01.1080p.web.h264-group.mkv] - yEnc (1/[PRiVATE] \\c2b510b594\\::686ea969999193.155368eba4965e56a8cd263382e012.f2712fdc::/97bd201cf931/) 1 (1/0)",
			0,
			"series.name.s01e01.1080p.web.h264-group.mkv",
		},
		{
			2,
			`[PRiVATE]-[WtFnZb]-[/More.Bla.S02E01.1080p.WEB.h264-EDITH[eztv.re].mkv-WtF[nZb]/More.Bla.S02E01.1080p.WEB.h264-EDITH.mkv]-[1/2] - "" yEnc  2990558544 (1/4173)`,
			1,
			"More.Bla.S02E01.1080p.WEB.h264-EDITH.mkv",
		},

		{
			0,
			`[011/116] - [AC-FFF] Highschool DxD BorN - 02 [BD][1080p-Hi10p] FLAC][Dual-Audio][442E5446].mkv yEnc (1/2401) 1720916370`,
			0,
			"[AC-FFF] Highschool DxD BorN - 02 [BD][1080p-Hi10p] FLAC][Dual-Audio][442E5446].mkv",
		},
		{
			0,
			`[010/108] - [SubsPlease] Ijiranaide, Nagatoro-san - 02 (1080p) [6E8E8065].mkv yEnc (1/2014) 1443366873`,
			0,
			"[SubsPlease] Ijiranaide, Nagatoro-san - 02 (1080p) [6E8E8065].mkv",
		},
		{
			0,
			`[1/8] - "TenPuru - No One Can Live on Loneliness v05 {+ "Book of Earthly Desires" pamphlet} (2021) (Digital) (KG Manga).cbz" yEnc (1/230) 164676947`,
			0,
			`TenPuru - No One Can Live on Loneliness v05 {+ "Book of Earthly Desires" pamphlet} (2021) (Digital) (KG Manga).cbz`,
		},
		{
			0,
			`[1/10] - "ONE.PIECE.S01E1109.1080p.NF.WEB-DL.AAC2.0.H.264-VARYG" yEnc (1/1277) 915318101`,
			0,
			`ONE.PIECE.S01E1109.1080p.NF.WEB-DL.AAC2.0.H.264-VARYG`,
		},
		{
			0,
			`[1/10] - "ONE.PIECE.S01E1109.1080p.NF.WEB-DL.AAC2.0.H.264-VARYG.mkv" yEnc (1/1277) 915318101`,
			0,
			`ONE.PIECE.S01E1109.1080p.NF.WEB-DL.AAC2.0.H.264-VARYG.mkv`,
		},
		{
			0,
			`[27/141] - "index.bdmv" yEnc (1/1) 280`,
			0,
			`index.bdmv`,
		},
	} {
		p := newSubjectParser(tc.fileCount)
		f := &File{Subject: tc.subject}
		p.Parse(f)
		if tc.fileCount != 0 {
			assert.Equal(t, tc.number, f.number)
		}
		assert.Equal(t, tc.name, f.name)
	}
}
