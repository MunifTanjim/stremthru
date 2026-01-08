package anidb

import (
	"encoding/xml"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type AnimeTitleDatasetItemTitle struct {
	XMLName xml.Name `xml:"title"`
	Type    string   `xml:"type,attr"` // main, official, syn, short, kana, card
	Lang    string   `xml:"lang,attr"`
	Value   string   `xml:",chardata"`
}

type AnimeTitleDatasetItem struct {
	XMLName xml.Name                     `xml:"anime"`
	AniDBId string                       `xml:"aid,attr"`
	Titles  []AnimeTitleDatasetItemTitle `xml:"title"`
}

func (a *AnimeTitleDatasetItem) Equal(b *AnimeTitleDatasetItem) bool {
	if a.AniDBId != b.AniDBId {
		return false
	}

	if len(a.Titles) != len(b.Titles) {
		return false
	}

	for i := range a.Titles {
		if a.Titles[i].Type != b.Titles[i].Type || a.Titles[i].Lang != b.Titles[i].Lang || a.Titles[i].Value != b.Titles[i].Value {
			return false
		}
	}

	return true
}

type detectedSeasonCountMap map[string]int

func (dscm detectedSeasonCountMap) Season() string {
	season := ""
	maxCount := 0
	for s, count := range dscm {
		if count > maxCount {
			season = s
			maxCount = count
		}
	}
	if season == "" {
		season = "1"
	}
	return season
}

func (dscm detectedSeasonCountMap) Count(season string) {
	if _, ok := dscm[season]; !ok {
		dscm[season] = 0
	}
	dscm[season] += 1
}

func SyncTitleDataset() error {
	log := logger.Scoped("anidb/dataset/titles")

	regexEnglishWithYear := regexp.MustCompile(`(?i) \(((?:19|20)\d{2})\)$`)
	regexEnglishWithS := regexp.MustCompile(`(?i) S(\d+)$`)
	regexWithSeason := regexp.MustCompile(`(?i):? \(?(?:Season|Saison|Stagione|Temporada|ภาคที่) (\d+)\)?\b`)
	regexWithOrdinalSuffixSeason := regexp.MustCompile(`(?i) (\d+)(?:st|nd|rd|th) Season\b`)
	regexPunctuation := regexp.MustCompile(`(?i)\p{P}`)
	regexWhitespaces := regexp.MustCompile(`(?i)\s+`)

	ds := util.NewXMLDataset(&util.XMLDatasetConfig[AnimeTitleDatasetItem, AnimeTitleDatasetItem]{
		DatasetConfig: util.DatasetConfig{
			Archive:     "gz",
			DownloadDir: path.Join(config.DataDir, "anidb-titles"),
			URL:         "https://anidb.net/api/anime-titles.xml.gz",
			DownloadHeaders: map[string]string{
				"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
			},
			Log: log,
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-24 * time.Hour))
			},
		},
		ListTagName: "animetitles",
		ItemTagName: "anime",
		Prepare: func(item *AnimeTitleDatasetItem) *AnimeTitleDatasetItem {
			slices.SortFunc(item.Titles, func(a, b AnimeTitleDatasetItemTitle) int {
				if r := strings.Compare(a.Type, b.Type); r != 0 {
					return r
				}
				return strings.Compare(a.Lang, b.Lang)
			})
			return item
		},
		GetItemKey: func(item *AnimeTitleDatasetItem) string {
			return item.AniDBId
		},
		IsItemEqual: func(a, b *AnimeTitleDatasetItem) bool {
			return a.Equal(b)
		},
		Writer: util.NewDatasetWriter(util.DatasetWriterConfig[AnimeTitleDatasetItem]{
			BatchSize: 200,
			Log:       log,
			Upsert: func(items []AnimeTitleDatasetItem) error {
				titles := []AniDBTitle{}
				for i := range items {
					item := &items[i]

					var mainTitle *AnimeTitleDatasetItemTitle
					var romajiSynTitle *AnimeTitleDatasetItemTitle
					var pinyinSynTitle *AnimeTitleDatasetItemTitle
					var englishTitle *AnimeTitleDatasetItemTitle
					itemTitles := []AniDBTitle{}
					detectedSeason := detectedSeasonCountMap{}
					year := ""

					seenAltMap := map[string]struct{}{}
					synAltIdx := 0

					for i := range item.Titles {
						shouldAppend := true

						t := &item.Titles[i]
						switch t.Type {
						case "main":
							title := *t
							mainTitle = &title
						case "official":
							if t.Lang == "en" {
								title := *t
								englishTitle = &title
							}

						case "syn":
							if t.Lang == "en" {
								if englishTitle == nil || (englishTitle.Type == t.Type && len(englishTitle.Value) < len(t.Value)) {
									title := *t
									englishTitle = &title
								}
							}

						default:
							shouldAppend = false
						}

						if year == "" {
							if match := regexEnglishWithYear.FindStringSubmatch(t.Value); len(match) > 1 {
								year = match[1]
							}
						}

						if t.Type != "short" {
							var seasonMatch []string
							if regexEnglishWithS.MatchString(t.Value) {
								seasonMatch = regexEnglishWithS.FindStringSubmatch(t.Value)
							} else if regexWithSeason.MatchString(t.Value) {
								seasonMatch = regexWithSeason.FindStringSubmatch(t.Value)
							} else if regexWithOrdinalSuffixSeason.MatchString(t.Value) {
								seasonMatch = regexWithOrdinalSuffixSeason.FindStringSubmatch(t.Value)
							}
							if len(seasonMatch) > 1 {
								season := seasonMatch[1]
								season = strings.TrimSpace(season)
								if season != "0" && !strings.HasPrefix(season, "0") && len(season) <= 2 {
									detectedSeason.Count(season)

									if t.Type == "syn" {
										switch t.Lang {
										case "x-jat":
											title := *t
											romajiSynTitle = &title
										case "x-zht":
											title := *t
											pinyinSynTitle = &title
										}
									}
								}
							}
						}

						if !shouldAppend {
							continue
						}

						key := t.Type + ":" + t.Lang
						if _, seen := seenAltMap[key]; seen {
							synAltIdx++
							t.Type = t.Type + "-alt-" + strconv.Itoa(synAltIdx)
						}
						seenAltMap[key] = struct{}{}

						itemTitles = append(itemTitles, AniDBTitle{
							TId:   item.AniDBId,
							TType: t.Type,
							TLang: t.Lang,
							Value: t.Value,
						})
					}
					for _, t := range []*AnimeTitleDatasetItemTitle{mainTitle, englishTitle, romajiSynTitle, pinyinSynTitle} {
						if t == nil {
							continue
						}

						title := AniDBTitle{
							TId:   item.AniDBId,
							TType: "clean-" + t.Type,
							TLang: t.Lang,
							Value: t.Value,
						}
						title.Value = regexWithOrdinalSuffixSeason.ReplaceAllLiteralString(regexWithSeason.ReplaceAllLiteralString(regexEnglishWithS.ReplaceAllLiteralString(regexEnglishWithYear.ReplaceAllLiteralString(title.Value, ""), ""), ""), "")
						title.Value = regexWhitespaces.ReplaceAllLiteralString(regexPunctuation.ReplaceAllLiteralString(title.Value, " "), " ")
						itemTitles = append(itemTitles, title)
					}
					season := detectedSeason.Season()
					for i := range itemTitles {
						t := &itemTitles[i]
						t.Season = season
						t.Year = year
					}
					titles = append(titles, itemTitles...)
				}
				return UpsertTitles(titles)
			},
			SleepDuration: 200 * time.Millisecond,
		}),
	})

	if err := ds.Process(); err != nil {
		return err
	}

	log.Info("rebuilding fts...")
	if err := RebuildTitleFTS(); err != nil {
		return err
	}

	return nil
}
