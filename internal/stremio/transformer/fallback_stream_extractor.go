package stremio_transformer

import (
	"regexp"
	"strings"
)

var codecPattern = `hevc|avc|mpeg|xvid|av1|x264|x265|h264|h265`
var qualityPattern = `\b(?:(?:blu.?ray|bd|br)[ .-]?(?:rip|remux)?|(?:web|dvd|sat|vhs|r5|scr)[ .-]?(?:dl|scr)?[ .-]?(?:mux|rip)?|(?:hc|(?:hd|pd)?tv)[ .-]?(?:rip|scr)?|(?:hd)?cam[ .-]?rip|(?:(?:tele)(?:sync|cine))|(?:hd[ .-]?)?(?:tc|ts))\b`

var codecRegex = regexp.MustCompile(`(?i)\b(` + codecPattern + `)\b`)
var qualityRegex = regexp.MustCompile(`(?i)\b(` + qualityPattern + `)\b`)
var resolutionRegex = regexp.MustCompile(`(?i)\b(\d{3,4}p|[248]k)\b`)
var sizeRegex = regexp.MustCompile(`(?i)\b([\d.]+ \w[bB])\b`)
var storeRegex = regexp.MustCompile(`(?i)\b(ad|dl|ed|oc|pp|pm|rd|tb|pkp|trb)\+?\b`)

func fallbackStreamExtractor(r *StreamExtractorResult) *StreamExtractorResult {
	input := r.Raw.Name
	if r.File.Name != "" {
		input += " " + r.File.Name
	}
	input += " " + r.Raw.Description

	println(input)
	if r.Codec == "" {
		if match := codecRegex.FindString(input); match != "" {
			r.Codec = match
		}
	}
	if r.Quality == "" {
		if match := qualityRegex.FindString(input); match != "" {
			r.Quality = match
		}
	}
	if r.Resolution == "" {
		if match := resolutionRegex.FindString(input); match != "" {
			r.Resolution = match
		}
	}
	if r.Size == "" {
		if match := sizeRegex.FindString(input); match != "" {
			r.Size = match
		}
	}
	if r.TTitle == "" {
		r.TTitle, _, _ = strings.Cut(r.Raw.Description, "\n")
	}
	if r.Store.Code == "" {
		if match := storeRegex.FindString(input); match != "" {
			if strings.HasPrefix(match, "+") {
				r.Store.Code = strings.TrimSuffix(match, "+")
				r.Store.IsCached = true
			} else {
				r.Store.Code = match
				if strings.Contains(input, "⚡️") {
					r.Store.IsCached = true
				}
			}
		}
	}
	return r
}
