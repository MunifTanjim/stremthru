package tvdb

import "strings"

type Award struct {
	Id        int    `json:"id"`
	Year      string `json:"year"`
	Details   any    `json:"details"`
	IsWinner  bool   `json:"isWinner"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	Series    any    `json:"series"`
	Movie     any    `json:"movie"`
	Episode   any    `json:"episode"`
	Character any    `json:"character"`
}

type Awards []Award

type awardsSummary struct {
	Wins        map[string][]string
	Nominations map[string][]string
}

func (awards Awards) Summary() awardsSummary {
	summary := awardsSummary{
		Wins:        map[string][]string{},
		Nominations: map[string][]string{},
	}
	for i := range awards {
		award := &awards[i]
		if award.IsWinner {
			summary.Wins[award.Name] = append(summary.Wins[award.Name], award.Category)
		} else {
			summary.Nominations[award.Name] = append(summary.Nominations[award.Name], award.Category)
		}
	}
	return summary
}

func (asummary awardsSummary) String() string {
	var result strings.Builder
	if len(asummary.Wins) > 0 {
		result.WriteString("[Won] ")
		idx := 0
		for name, categories := range asummary.Wins {
			if idx > 0 {
				result.WriteString(", ")
			}
			idx++
			result.WriteString(name)
			result.WriteString(" (")
			for i, category := range categories {
				if i > 0 {
					result.WriteString(", ")
				}
				result.WriteString(category)
			}
			result.WriteString(")")
		}
		result.WriteString(".")
	}
	if len(asummary.Nominations) > 0 {
		if len(asummary.Wins) > 0 {
			result.WriteString(" \n")
		}
		result.WriteString("[Nominated] ")
		for name, categories := range asummary.Nominations {
			result.WriteString(name)
			result.WriteString(" (")
			for i, category := range categories {
				if i > 0 {
					result.WriteString(", ")
				}
				result.WriteString(category)
			}
			result.WriteString(")")
		}
		result.WriteString(".")
	}
	return result.String()
}
