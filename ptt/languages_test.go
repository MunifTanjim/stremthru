package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguages(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ttitle    string
		languages []string
	}{
		{"russian", "Deadpool 2016 1080p BluRay DTS Rus Ukr 3xEng HDCL", []string{"russian", "ukrainian"}},
		{"netherlands", "VAIANA: MOANA (2017) NL-Retail [2D] EAGLE", []string{"dutch"}},
		{"flemish", "De Noodcentrale S02E05 FLEMISH 540p WEBRip AAC H 264", []string{"dutch"}},
		{"truefrench", "The Intern 2015 TRUEFRENCH 720p BluRay x264-PiNKPANTERS", []string{"french"}},
		{"vff", "After Earth 2013 VFF BDrip x264 YJ", []string{"french"}},
		{"french", "127.Heures.FRENCH.DVDRip.AC3.XViD-DVDFR", []string{"french"}},
		{"vostfr", "Color.Of.Night.Unrated.DC.VostFR.BRrip.x264", []string{"french"}},
		{"multi language", "Le Labyrinthe 2014 Multi-VF2 1080p BluRay x264-PopHD", []string{"multi audio"}},
		{"VFI", "Maman, j'ai raté l'avion 1990 VFI 1080p BluRay DTS x265-HTG", []string{"french"}},
		{"italian", "South.Park.S21E10.iTALiAN.FiNAL.AHDTV.x264-NTROPiC", []string{"italian"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}

	for _, tc := range []struct {
		ttitle    string
		languages []string
	}{
		{"2- English- {SDH}.srt", []string{"english"}},
		{"1_ English-Subs.srt", []string{"english"}},
		{"House S 1 CD 1-6 svensk, danska, norsk, finsk sub", []string{"danish", "finnish", "swedish", "norwegian"}},
		{"Svein.Og.Rotta.NORSK.Nordic.Subs.2006", []string{"danish", "finnish", "swedish", "norwegian"}},
		{"Borat med Norsk Undertekst", []string{"norwegian"}},
		{"Subs/21_Bokmal.srt", []string{"norwegian"}},
		{"Subs/nob.srt", []string{"norwegian"}},
		{"Subs/5_nor.srt", []string{"norwegian"}},
		{"Curious.George.2.Follow.That.Monkey.2009.DK.SWE.UK.PAL.DVDR-CATC", []string{"danish", "swedish"}},
		{"Yes.Man.Dk-Subs.2009.dingel", []string{"danish"}},
		{"Dan-SDH.srt", []string{"danish"}},
		{"dan.srt", []string{"danish"}},
		{"Red Riding 1974 [2009 PAL DVD][En Subs[Sv.No.Fi]", []string{"english", "finnish", "swedish"}},  // TODO: does not include no
		{"Comme une Image (Look at Me) [2004 PAL DVD][Fr Subs[Sv.Da.No]", []string{"french", "swedish"}}, // TODO: does not include da,no
		{"A.Good.Day.To.Die.Hard.2013.SWESUB.DANSUB.FiNSUB.720p.WEB-DL.-Ro", []string{"danish", "finnish", "swedish"}},
		{"The.Prisoner.1967-1968.Complete.Series.Subs.English+Nordic", []string{"english", "danish", "finnish", "swedish", "norwegian"}},
		{"Royal.Pains.S05E02.HDTV.subtitulado.esp.sc.avi", []string{"spanish"}},
		{"Desmembrados (2006) [HDrip-XviD-AC3][Castellano]", []string{"spanish"}},
		{"10_Spanish-Subs.srt", []string{"spanish"}},
		{"Patriot Games [1992] Eng, Ger, Cze, Hun, Pol + multisub  DVDrip", []string{"multi subs", "english", "german", "polish", "czech", "hungarian"}},
		{"Elvis Presley - La via del Male (King creole) - IT EN FR DE ES", []string{"english", "french", "spanish", "italian", "german"}},
		{"FernGully [H264 - Ita Dut Fre Ger Eng Spa Aac - MultiSub]", []string{"multi subs", "english", "french", "spanish", "italian", "german", "dutch"}},
		{"Jesus de Montreal / Jesus of Montreal - subtitulos espanol", []string{"spanish"}},
		{"Los.Vengadores.DVDR español ingles clon", []string{"english", "spanish"}},
		{"EMPIRE STATE 2013 DVDRip TRNC English and Española Latin", []string{"english", "latino", "spanish"}},
		{"Mary Poppins Returns 2019 DVDRip LATINO-1XBET", []string{"latino"}},
		{"Los Simpsons S18 E01 Latino", []string{"latino"}},
		{"Spider-Man (2002) Blu-Ray [720p] Dual Ingles-Español", []string{"dual audio", "english", "spanish"}},
		{"Abuela (2015) 1080p BluRay x264 AC3 Dual Latino-Inglés", []string{"dual audio", "english", "latino"}},
		{"Dumbo.2019.1080p.Dual.Lat", []string{"dual audio", "latino"}},
		{"A.Score.to.Settle.2019.lati.mp4", []string{"latino"}},
		{"[S0.E04] Gambit królowej - Gra środkowa.Spanish Latin America.srt", []string{"latino"}},
		{"Latin American.spa.srt", []string{"latino"}},
		{"Men in Black International 2019 (inglês português)", []string{"english", "portuguese"}},
		{"Assassins (1995) Sylvester Stallone.DVDrip.XviD - Italian Englis", []string{"english", "italian"}},
		{"El Club de la Lucha[dvdrip][spanish]", []string{"spanish"}},
		{"The Curse Of The Weeping Woman 2019 BluRay 1080p Tel+Tam+hin+eng", []string{"english", "hindi", "telugu", "tamil"}},
		{"Inception 2010 1080p BRRIP[dual-audio][eng-hindi]", []string{"dual audio", "english", "hindi"}},
		{"Inception (2010) 720p BDRip Tamil+Telugu+Hindi+Eng", []string{"english", "hindi", "telugu", "tamil"}},
		{"Subs/Dear.S01E02.WEBRip.x265-ION265/34_tam.srt", []string{"tamil"}},
		{"Subs/Dear.S01E02.WEBRip.x265-ION265/35_tel.srt", []string{"telugu"}},
		{"Dabangg 3 2019 AMZN WebRip Hindi 720p x264", []string{"hindi"}},
		{"Quarantine [2008] [DVDRiP.XviD-M14CH0] [Lektor PL] [Arx]", []string{"polish"}},
		{"The Mandalorian S01E06 POLISH WEBRip x264-FLAME", []string{"polish"}},
		{"Na Wspólnej (2024) [E3951-3954][1080p][WEB-DL][PL][x264-GhN]", []string{"polish"}},
		{"Kulej.2024.PL.1080p.AMZN.WEB-DL.x264-KiT", []string{"polish"}},
		{"Star.Wars.Skeleton.Crew.Sezon01.PLDUB.480p.DSNP.WEB-DL.H264.DDP5.1-K83", []string{"polish"}},
		{"Bukmacher / Bookie [E02E07] [MULTi] [1080p] [AMZN] [WEB-DL] [H264] [DDP5.1.Atmos-K83] [Lektor PL]", []string{"multi audio", "polish"}},
		{"Gra 1968 [REKONSTRUKCJA] [1080p.WEB-DL.H264.AC3-FT] [Film Polski]", []string{"polish"}},
		{"Evil.Dead.Rise.2023.PLSUB.720p.MA.WEB-DL.H264.E-AC3-CMRG", []string{"polish"}},
		{"Wallace i Gromit: Zemsta pingwina / Wallace & Gromit: Vengeance Most Fowl (2024) [480] [WEB-DL] [XviD] [DD5.1-K83] [Dubbing PL]", []string{"polish"}},
		{"Strażniczka smoków / Dragonkeeper (2024) [1080p] [H264] [Napisy PL]", []string{"polish"}},
		{"Carros 2 Dublado - Portugues BR (2011)", []string{"portuguese"}},
		{"A.Simples.Plan.1998.720p.BDRIP.X264.dublado.portugues.BR.gmenezes", []string{"portuguese"}},
		{"American.Horror.Story.S01E01.720p. PORTUGUÊS BR", []string{"portuguese"}},
		{"Angel.S05E19.legendado.br.rmvb", []string{"portuguese"}},
		{"Grimm S01E11 Dublado BR [ kickUploader ]", []string{"portuguese"}},
		{"InuYasha.EP161.ptBR.subtitles.[inuplace.com.br].avi", []string{"portuguese"}},
		{"Ghost.Rider.DivX_Gamonet(Ingles-Port.BR)-AC3.avi", []string{"english", "portuguese"}},
		{"I Am David  legendado pt/br", []string{"portuguese"}},
		{"Lone Wolf and Cub 6 movies - legendas BR", []string{"portuguese"}},
		{"Wonder Woman Season 3 (H.264 1080p; English/Portuguese-BR)", []string{"english", "portuguese"}},
		{"MIB 3 - Homens de Preto 2012 ( Audio EN-BR - Leg BR  Mkv 1280x69", []string{"english", "portuguese"}},
		{"my wife is a gangster 3 legendado em PT(BR)", []string{"portuguese"}},
		{"A.Clockwork.Orange.1971.BRDRIP.1080p.DUAL.PORT-BR.ENG.gmenezes.m", []string{"dual audio", "english", "portuguese"}},
		{"Superman I - O Filme 1978 Leg. BR - Mkv 1280x528", []string{"portuguese"}},
		{"Subs/Brazilian.por.srt", []string{"portuguese"}},
		{"Brazilian Portuguese.por.srt", []string{"portuguese"}},
		{"[S0.E07] Gambit królowej - Gra koncowa.Portuguese Brazil.srt", []string{"portuguese"}},
		{"The Hit List (2011) DVD NTSC WS (eng-fre-pt-spa) [Sk]", []string{"english", "french", "spanish"}}, // TODO: does not include pt
		{"[POPAS] Neon Genesis Evangelion: The End of Evangelion [jp_PT-pt", []string{"japanese", "portuguese"}},
		{"Zola Maseko - Drum (2004) PT subs", []string{"portuguese"}},
		{"Idrissa Ouedraogo - Yaaba (1989) EN ES FR PT", []string{"english", "french", "spanish"}}, // TODO: does not include pt
		{"Metallica.Through.The.Never.2013 O Filme(leg.pt-pt)", []string{"portuguese"}},
		{"Dinossauro (2000) --[ Ing / Pt / Esp ]", []string{"english", "spanish"}}, // TODO: does not include pt
		{"Mulan 1 (1998) Versao Portuguesa", []string{"portuguese"}},
		{"The Guard 2011.DK.EN.ES.HR.NL.PT.RO.Subtitles", []string{"english", "spanish", "romanian", "croatian", "dutch", "danish"}},
		{"Titan.A.E.2000 720p  HDTV DTS Eng Fra Hun Rom Rus multisub", []string{"multi subs", "english", "french", "russian", "hungarian", "romanian"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.Latin American es.ass", []string{"latino"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.Brazilian pt.ass", []string{"portuguese"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.pt.ass", []string{"portuguese"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.es.ass", []string{"spanish"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.de.ass", []string{"german"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.it.ass", []string{"italian"}},
		{"Frieren - Beyond Journey's End - S01E01 - TBA WEBDL-1080p.ar.ass", []string{"arabic"}},
		{"Subs(ara,fre,ger).srt", []string{"french", "german", "arabic"}},
		{"Subs(chi,eng,ind,kor,may,tha,vie).srt", []string{"english", "korean", "chinese", "vietnamese", "indonesian", "thai", "malay"}},
		{"Miami.Bici.2020.1080p.NETFLIX.WEB-DL.DDP5.1.H.264.EN-ROSub-ExtremlymTorrents", []string{"english", "romanian"}},
		{"26_Romanian.srt", []string{"romanian"}},
		{"4_Bulgarian.srt", []string{"bulgarian"}},
		{"18_srp.srt", []string{"serbian"}},
		{"Aranyelet.S01.HUNGARIAN.1080p.WEBRip.DDP5.1.x264-SbR[rartv]", []string{"hungarian"}},
		{"Ponyo[2008]DvDrip-H264 Quad Audio[Eng Jap Fre Spa]AC3 5.1[DXO]", []string{"english", "japanese", "french", "spanish"}},
		{"The Mechanic [1972] Eng,Deu,Fra,Esp,Rus + multisub DVDrip", []string{"multi subs", "english", "french", "spanish", "german", "russian"}},
		{"Mommie Dearest [1981 PAL DVD][En.De.Fr.It.Es Multisubs[18]", []string{"multi subs", "english", "french", "spanish", "german"}}, // TODO: does not include it
		{"Pasienio sargyba S01E03 (2016 WEBRip LT)", []string{"lithuanian"}},
		{"24_Lithuanian.srt", []string{"lithuanian"}},
		{"25_Latvian.srt", []string{"latvian"}},
		{"13_Estonian.srt", []string{"estonian"}},
		{"Ip.Man.4.The.Finale.2019.CHINESE.1080p.BluRay.x264.TrueHD.7.1.Atmos-HDC", []string{"chinese"}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}

	for _, tc := range []struct {
		name      string
		ttitle    string
		languages []string
	}{
		{"CHT", "[NC-Raws] 叫我對大哥 (WEB版) / Ore, Tsushima - 10 [Baha][WEB-DL][1080p][AVC AAC][CHT][MP4]", []string{"chinese"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}

	for _, tc := range []struct {
		ttitle    string
		languages []string
	}{
		{"Inuyasha_TV+Finale+OVA+Film+CD+Manga+Other; dub jpn,chn,eng sub chs (2019-09-21)", []string{"english", "japanese", "chinese"}},
		{"Initial D Live Action 2005 ENG/CHI", []string{"english", "chinese"}},
		{"Wolf.Warrior.2015.720p.BluRay.x264.Mandarin.AAC-ETRG", []string{"chinese"}},
		{"9_zh-Hans.srt", []string{"chinese"}},
		{"Traditional Chinese.chi.srt", []string{"taiwanese"}},
		{"Subs/Promare - Chinese (Traditional).ass", []string{"taiwanese"}},
		{"10_zh-Hant.srt", []string{"taiwanese"}},
		{"Berserk 01-25 [dual audio JP,EN] MKV", []string{"dual audio", "english", "japanese"}},
		{"FLCL S05.1080p HMAX WEB-DL DD2.0 H 264-VARYG (FLCL: Shoegaze Dual-Audio Multi-Subs)", []string{"multi subs", "dual audio"}},
		{"Shinjuku Swan 2015 JAP 1080p BluRay x264 DTS-JYK", []string{"japanese"}},
		{"Wet.Woman.in.the.Wind.2016.JAPANESE.1080p.BluRay.x264-iKiW", []string{"japanese"}},
		{"All.Love.E146.KOR.HDTV.XViD-DeBTV", []string{"korean"}},
		{"The.Nun.2018.KORSUB.HDRip.XviD.MP3-STUTTERSHIT", []string{"korean"}},
		{"Burning.2018.KOREAN.720p.BluRay.H264.AAC-VXT", []string{"korean"}},
		{"Atonement.2017.KOREAN.ENSUBBED.1080p.WEBRip.x264-VXT", []string{"english", "korean"}},
		{"A Freira (2018) Dublado HD-TS 720p", []string{"portuguese"}},
		{"Escobar El Patron Del Mal Capitulo 91 SD (2012-10-10) [SiRaDuDe]", []string{"portuguese"}},
		{"Bleach - 215 ao 220 - [DB-BR]", []string{"portuguese"}},
		{"Joker.2019.MULTi.Bluray.1080p.Atmos.7.1.En.Fr.Sp.Pt-DDR[EtHD]", []string{"multi audio", "english", "french"}}, // TODO: does not include sp,pt
		{"Dilbert complete series + en subs", []string{"english"}},
		{"The Next Karate Kid (1994) NTSC WS -Eng/Fre/Spa/Por- [ctang]", []string{"english", "french", "spanish", "portuguese"}},
		// {"arsenico por compasion 1944 Capra spanish castellano", []string{"spanish"}}, // TODO: should not detect
		{"Un.Homme.Et.Une.Femme.1966.DVDRip.XviD.AR [PT ENG ESP]", []string{"english", "spanish", "portuguese"}},
		{"Geminis.2005.Argentina.ESP.ENG.PT.SUBS", []string{"english", "spanish", "portuguese"}},
		{"1917 2019 1080p Bluray x264-Sexmeup [Greek Subs] [Braveheart]", []string{"greek"}},
		{"The Lion King 1,2,3 Greek Language", []string{"greek"}},
		{"The Adams Family (1991) (Greek Subs integratet)", []string{"greek"}},
		{"6_Greek.srt", []string{"greek"}},
		{"The Insult (2017) [BluRay] [720p] Arabic", []string{"arabic"}},
		{"The.Mexican.2001 - Arabic Subs Hardcoded", []string{"arabic"}},
		{"Much Loved (2015) - DVDRip x265 HEVC - ARAB-ITA-FRE AUDIO (ENG S", []string{"english", "french", "italian", "arabic"}},
		{"42.2013.720p.BluRay.x264.HD4Ar Arab subtitle", []string{"arabic"}},
		{"Fauda.S01.HEBREW.1080p.NF.WEBRip.DD5.1.x264-TrollHD[rartv]", []string{"hebrew"}},
		{"madagascar 720p hebrew dubbed.mkv", []string{"hebrew"}},
		{"Into.the.Night.S01E04.Ayaz.1080p.NF.WEB-DL.DDP5.1.x264-NTG_track17_[heb].srt", []string{"hebrew"}},
		{"The.Protector.2018.S03.TURKISH.WEBRip.x264-ION10", []string{"turkish"}},
		{"Recep Ivedik 6 (2020) NETFLIX 720p WEBDL (Turkish) - ExtremlymTorrents", []string{"turkish"}},
		{"The Insider*1999*[DVD5][PAL][ENG, POL, sub. ROM, TUR]", []string{"english", "polish", "romanian", "turkish"}},
		{"Divorzio allitaliana [XviD - Ita Mp3 - Sub Eng Esp Tur]", []string{"english", "spanish", "italian", "turkish"}},
		{"2007 Saturno Contro [Saturn In Opposition] (ITA-FRA-TUR) [EngSub", []string{"english", "french", "italian", "turkish"}},
		{"Cowboy Bebop - 1080p BDrip Audio+sub MULTI (VF / VOSTFR)", []string{"multi audio", "french"}},
		{"Casablanca 1942 BDRip 1080p [multi language,multi subs].mkv", []string{"multi subs", "multi audio"}},
		{"Avengers.Endgame.2019.4K.UHD.ITUNES.DL.H265.Dolby.ATMOS.MSUBS-Deflate.Telly", []string{"multi subs"}},
		{"Dawn of the Planet of the Apes (2014) 720p BluRay x264 - MULTI S", []string{"multi subs"}},
		{"Pirates of the Caribbean On Stranger Tides (2011) DVD5 (Multi Su", []string{"multi subs"}},
		{"Jumanji The Next Level (2019) 720p HDCAM Ads Blurred x264 Dual A", []string{"dual audio"}},
		{"Men in Black International (2019) 720p Korsub HDRip x264 ESub [Dual Line Audio] [Hindi English]", []string{"dual audio", "english", "korean", "hindi"}},
		{"Fame (1980) [DVDRip][Dual][Ac3][Eng-Spa]", []string{"dual audio", "english", "spanish"}},
		{"O Rei do Show 2018 Dual Áudio 4K UtraHD By.Luan.Harper", []string{"dual audio"}},
		{"Cowboy Bebop The Movie (2001) BD 1080p.x265.Tri-Audio.Ita.Eng.Jap [Rady]", []string{"multi audio", "english", "japanese", "italian"}},
		{"[IceBlue] Naruto (Season 01) - [Multi-Dub][Multi-Sub][Dublado][HEVC 10Bits] 800p BD", []string{"multi subs", "multi audio", "portuguese"}},
		{"Blue Seed - 01 (BDRip 720x480p x265 HEVC FLAC, AC3x2 2.0x3)(Triple Audio)[sxales].mkv", []string{"multi audio"}},
		{"Ministri S01E02 SLOVAK 480p x264-mSD", []string{"slovakian"}},
		{"Subs/35_slo.srt", []string{"slovakian"}},
		{"Seinfeld.COMPLETE.SLOSUBS.DVDRip.XviD", []string{"slovakian"}},
		{"Subs/36_Slovenian.srt", []string{"slovenian"}},
		{"The House Bunny (2008) BDRemux 1080p MediaClub [RUS, UKR, ENG]", []string{"english", "russian", "ukrainian"}},
		{"L'immortel (2010) DVDRip AVC (Russian,Ukrainian)", []string{"russian", "ukrainian"}},
		{"Into.the.Night.S01E03.Mathieu.1080p.NF.WEB-DL.DDP5.1.x264-NTG_track33_[vie].srt", []string{"vietnamese"}},
		{"Subs/vie.srt", []string{"vietnamese"}},
		{"Subs/Vietnamese.srt", []string{"vietnamese"}},
		{"Subs/Dear.S01E05.WEBRip.x265-ION265/25_may.srt", []string{"malay"}},
		{"Midnight.Diner.Tokyo.Stories.S02E10.WEBRip.x264-ION10/14_Indonesian.srt", []string{"indonesian"}},
		{"Inglês,Português,Italiano,Francês,Polonês,Russo,Norueguês,Dinamarquês,Alemão,Espanhol,Chinês,Japonês,Coreano,Persa,Hebraico,Sueco,Árabe,Holandês,Tâmil,Tailandês", []string{"english", "japanese", "korean", "chinese", "french", "spanish", "portuguese", "italian", "german", "russian", "tamil", "polish", "dutch", "danish", "swedish", "norwegian", "thai", "hebrew", "persian"}},
		{"russian,english,ukrainian", []string{"english", "russian", "ukrainian"}},
		{"Subs/Thai.srt", []string{"thai"}},
		{"Food Choices (2016) WEB.1080p.H264_tha.srt", []string{"thai"}},
		{"Ekk Deewana Tha (2012) DVDRip 720p x264 AAC TaRa.mkv", nil},
		{"My Big Fat Greek Wedding (2002) 720p BrRip x264 - YIFY", nil},
		{"Get Him to the Greek 2010 720p BluRay", nil},
		{"[Hakata Ramen] Hoshiai No Sora (Stars Align) 01 [1080p][HEVC][x265][10bit][Dual-Subs] HR-DR", nil},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}

	for _, tc := range []struct {
		name      string
		ttitle    string
		languages []string
	}{
		{"not detect LT language from yts domain name", "Do.Or.Die.1991.1080p.BluRay.x264-[YTS.LT].mp4", nil},
		{"not detect PT language from temporada season naming", "Castlevania 2017 1º temporada completa 1080p", []string{"spanish"}},
		{"not detect PT language with cap. episode title", "City on a Hill - Temporada 1 [HDTV][Cap.110].avi", []string{"spanish"}},
		{"not detect NL language from website", "La inocencia [720][wWw.EliteTorrent.NL].mkv", nil},
		{"not detect FI language from website", "Reasonable Doubt - Temporada 1 [HDTV][Cap.101][www.AtomoHD.FI]", []string{"spanish"}},
		{"not detect FI language from website v2", "1883 - Temporada 1 [HDTV 720p][Cap.103][AC3 5.1][www.AtomoHD.fi]", []string{"spanish"}},
		{"not detect TW language from website", "Los Winchester - Temporada 1 [HDTV][Cap.103][www.atomoHD.tw]", []string{"spanish"}},
		{"not detect CH language from website", "El Inmortal- Temporada 1 [HDTV 720p][Cap.104][AC3 5.1][www.AtomoHD.ch]]", []string{"spanish"}},
		{"not detect TEL language from website", "Black Friday (2021) [BluRay Rip][AC3 5.1][www.atomixHQ.TEL]", nil},
		{"not detect SE language from website", "Deep Blue Sea 3 [HDR][wWw.EliteTorrent.SE]", nil},
		{"not detect language from title before year", "The.Italian.Job.1969.1080p.BluRay.x265-RARBG.mp4", nil},
		{"not detect language from title before year v2", "Chinese Zodiac (2012) 1080p BrRip x264 - YIFY", nil},
		{"not detect language from title before year v3", "The German Doctor 2013 1080p WEBRip", nil},
		{"not detect language from title before year v4", "Johnny English 2003 1080p BluRay", nil},
		{"not detect language from title before year v5", "Polish Wedding (1998) 1080p (moviesbyrizzo upl).mp4", nil},
		{"not detect language from title before year v6", "Russian.Doll.S02E02.2160p.NF.WEB-DL.DDP5.1.HDR.DV.HEVC-PEXA.mkv", nil},
		{"not detect language from title before year v7", "The.Spanish.Prisoner.1997.1080p.BluRay.x265-RARBG", nil},
		{"not detect language from title before year v8", "Japanese.Story.2003.1080p.WEBRip.x264-RARBG", nil},
		{"not detect language from title before year v9", "[ Torrent9.cz ] The.InBetween.S01E10.FiNAL.HDTV.XviD-EXTREME.avi", nil},
		{"not detect language from title before year v10", "Thai Massage (2022) 720p PDVDRip x264 AAC.mkv", nil},
		{"not detect dan language", "Carson Daly 2017 09 13 Dan Harmon 720p HDTV x264-CROOKS", nil},
		{"not detect dan language v2", "Dan Browns The Lost Symbol S01E03 1080p WEB H264-GLHF", nil},
		{"not detect ara language", "Ben.Ara.2015.1080p.WEBRip.x265-RARBG.mp4", nil},
		{"not detect ara language v2", "Ara.(A.Break).2008.DVDRip", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}

	t.Run("not remove english from title", func(t *testing.T) {
		result := Parse("The English Patient (1996) 720p BrRip x264 - YIFY")
		assert.Empty(t, result.Languages)
		assert.Equal(t, "The English Patient", result.Title)
	})

	// PY
	for _, tc := range []struct {
		ttitle    string
		languages []string
	}{
		{"www.1TamilMV.cz - Mirzapur (2020) S02 EP (01-10) - HQ HDRip - [Tam+ Tel] - x264 - AAC - 1GB - ESub", []string{"english", "telugu", "tamil"}},
		{"www.1TamilMV.cz - The Game of Chathurangam (2023) WEB-DL - 1080p - AVC - (AAC 2.0) [Tamil + Malayalam] - 1.2GB.mkv", []string{"tamil", "malayalam"}},
		{"www.1TamilMV.yt - Anchakkallakokkan (2024) Malayalam HQ HDRip - 720p - HEVC - (DD+5.1 - 192Kbps & AAC) - 750MB - ESub.mkv", []string{"english", "malayalam"}},
		{"Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano][www.AtomoHD.nu].avi", []string{"spanish"}},
		{"Godzilla.x.Kong.The.New.Empire.2024.2160p.BluRay.REMUX.DV.P7.HDR.ENG.LATINO.GER.ITA.FRE.HINDI.CHINESE.TrueHD.Atmos.7.1.H265-BEN.THE.MEN", []string{"english", "chinese", "french", "latino", "italian", "german", "hindi"}},
		{"Sampurna.2023.Bengali.S02.1080p.AMZN.WEB-DL.DD+2.0.H.265-TheBiscuitMan", []string{"bengali"}},
		{"Kingdom.of.the.Planet.of.the.Apes.2024.HDRIP.1080P.[xDark [SaveHD] Latin + English + Hindi.mp4", []string{"english", "latino", "hindi"}},
		{"The Karate Kid Part III 1989 1080p DUAL TİVİBU WEB-DL x264 AAC - HdT", []string{"dual audio", "turkish"}},
		{"The French Connection 1971 Remastered BluRay 1080p REMUX AVC DTS-HD MA 5 1-LEGi0N", nil},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.languages, result.Languages)
		})
	}
}
