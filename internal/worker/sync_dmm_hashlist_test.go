package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDMMHashlistItems(t *testing.T) {
	repoDir := "testdata/dmm_hashlist"

	for _, tc := range []struct {
		name     string
		filename string
		items    []DMMHashlistItem
		err      string
	}{
		{"inline wrapped", "00122b10-c057-4040-bcf1-b4b1cdb238f1.html", []DMMHashlistItem{
			{"[4k世界]姥姥的外孙4k.How.to.Make.Millions.Before.Grandma.Dies.2024.2160p.NF.WEB-DL.DDP5.1.H.265-4ksj.net", "46402db82f015ec87eef5b25439395d4ea54df01", 13204795128},
			{"Я, Алекс Кросс.2012.Blu-Ray.Remux.(1080p).mkv", "936bd501035ac58a7170912d0b658faaeb13213e", 30802205238},
		}, ""},
		{"inline array", "00011863-7c82-4d3c-846a-deeb5ac0d6be.html", []DMMHashlistItem{
			{"Yu-Gi-Oh! Arc-V Amazon Web-DL", "cbcac562c4aa0f2b3588464ffc9a088c737f024c", 188826186661},
			{"Yu-Gi-Oh! Zexal Amazon Web-DL", "af31ca6371b93a265948eda9742f3ec6fef20746", 266679583103},
		}, ""},
		{"meta refresh", "37b9c4dd-8ea7-4a67-9990-ef110f19a6bc.html", []DMMHashlistItem{
			{"The.Wire.S03.1080p.BluRay.x264-ROVERS[rartv]", "e63ca39d760e0433bc1ce6bde0f0aade33e7a9d3", 57042973952},
			{"The.Wire.S04.1080p.BluRay.x264-ROVERS[rartv]", "c92f06b763e90df8b323d6a9ba2a6958346df7c6", 64049234096},
		}, ""},
		{"stored raw", "15dc99e3-39d1-4f86-bfc7-fc3ef59300ae.html", []DMMHashlistItem{
			{"Sherlock.S04E01.The.Six.Thatchers.1080p.HEVC.x265-MeGusta.mkv", "a96a231ba49d402737114499de94a67304ab8645", 1006072466},
			{"Vera.S01E01-04.WEBDL.1080p.ITA", "60830ed2f6ee04f6cda4b1a81b0c292b9db4b2e2", 13633227982},
		}, ""},
		{"stored page", "007eb1d2-7272-49ce-a659-ccc2021baafb.html", []DMMHashlistItem{
			{"Outback Opal Hunters S11 E09-E11 1080p WEBRip x264-skorpion", "e4c34250c848d6fa72089207da3a1f4082a15f3b", 5445078451},
			{"Outback Opal Hunters S11 E18-E20 1080p WEBRip x264-skorpion", "d5ad4ad4302182d010e136692938e105a574b56d", 5126135378},
		}, ""},
		{"empty fragment", "00000000-0000-0000-0000-000000000001.html", nil, ""},
		{"invalid stored id", "00000000-0000-0000-0000-000000000002.html", nil, "invalid stored hashlist id"},
		{"missing stored list", "00000000-0000-0000-0000-000000000003.html", nil, "failed to read stored hashlist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, err := extractDMMHashlistItems(repoDir, tc.filename)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.items, items)
		})
	}
}
