package nntp

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// validateInput checks for command injection attempts
func validateInput(s string) error {
	if strings.ContainsAny(s, "\r\n") {
		return NewProtocolError(0, "invalid input: contains CR/LF characters")
	}
	return nil
}

// parses CAPABILITIES response
func parseCapabilities(lines []string) (*Capabilities, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty CAPABILITIES response")
	}

	caps := &Capabilities{
		Capabilities: make([]string, 0, len(lines)),
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "VERSION ") {
			caps.Version = strings.TrimPrefix(line, "VERSION ")
		}
		caps.Capabilities = append(caps.Capabilities, line)
	}

	return caps, nil
}

// parses LIST ACTIVE.TIMES response
// Format: "group timestamp creator"
func parseActiveTimes(lines []string) ([]NewsGroupActiveTime, error) {
	groups := make([]NewsGroupActiveTime, 0, len(lines))

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		groups = append(groups, NewsGroupActiveTime{
			Name:      parts[0],
			CreatedAt: time.Unix(timestamp, 0),
			Creator:   parts[2],
		})
	}

	return groups, nil
}

// parses LIST NEWSGROUPS response
// Format: "group description"
func parseNewsGroups(lines []string) ([]NewsGroup, error) {
	newsgroups := make([]NewsGroup, 0, len(lines))

	for _, line := range lines {
		name, description, _ := strings.Cut(line, " ")
		newsgroups = append(newsgroups, NewsGroup{
			Name:        name,
			Description: strings.TrimSpace(description),
		})
	}

	return newsgroups, nil
}

// parseDistribPats parses LIST DISTRIB.PATS response
// Format: "weight:wildmat:header"
func parseDistribPats(lines []string) ([]DistribPat, error) {
	pats := make([]DistribPat, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		weight, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		pats = append(pats, DistribPat{
			Weight:  weight,
			Wildmat: parts[1],
			Header:  parts[2],
		})
	}

	return pats, nil
}
