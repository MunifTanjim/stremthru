package nzb

import (
	"regexp"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/util"
)

var (
	quotedFilenameRegex                        = regexp.MustCompile(`"([^"].+[^"])"`)
	bracketedFilenameRegex                     = regexp.MustCompile(`\[(?:(?:(?:[^\/\[\]]*(?:\[[^\]]*\])?)*)?\/)*([^\[\]]*(?:\[[^\]]*\])?[^\[\/\.]*\.\w{2,5})\]`)
	fileCountFilenameYencSegmentCountSizeregex = regexp.MustCompile(`^[\[\(]\d+\/\d+[\]\)]\s-\s(.*)\syEnc\s[\[\(]\d+\/\d+[\]\)]\s\d+`)
	fileCountFilenameRegex                     = regexp.MustCompile(`[\[\(]\d+\/\d+[\]\)]\s-\s(.*\.\w{2,5})`)
	reFilenameSegmentCountRegex                = regexp.MustCompile(`^Re:\s*(.+\.\w{2,5})(?: [\[\(]\d+\/\d+[\]\)])?`)
	somethingHashLikeRegex                     = regexp.MustCompile(`\\[A-Z0-9]+\\::(.+)\s+yEnc\s+(?:[\[\(]\d+\/\d+[\)\]])?`)
	likeFilenameRegex                          = regexp.MustCompile(`\b([\w\-+()' .,]+(?:\[[\w\-/+()' .,]*][\w\-+()' .,]*)*\.[A-Za-z0-9]{2,4})\b`)
)

type subjectParser struct {
	fileCount      int
	fileIndexRegex *regexp.Regexp
}

func newSubjectParser(fileCount int) *subjectParser {
	fileIndexRegex := regexp.MustCompile(`[(\[]\s*(\d+)\s*/\s*0?` + util.IntToString(fileCount) + `\s*[\])]`)
	p := subjectParser{
		fileCount:      fileCount,
		fileIndexRegex: fileIndexRegex,
	}
	return &p
}

func (p *subjectParser) Parse(f *File) {
	subject := f.Subject

	if f.name == "" {
		if matches := quotedFilenameRegex.FindStringSubmatch(subject); len(matches) == 2 {
			if name := strings.TrimSpace(matches[1]); name != "" {
				f.name = name
				subject = strings.TrimSpace(strings.Replace(subject, matches[0], "", 1))
			}
		}
	}

	if f.name == "" {
		if matches := bracketedFilenameRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[0], "", 1))
		}
	}

	if f.name == "" {
		if matches := fileCountFilenameYencSegmentCountSizeregex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[0], "", 1))
		}
	}

	if f.name == "" {
		if matches := fileCountFilenameRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[1], "", 1))
		}
	}

	if f.name == "" {
		if matches := reFilenameSegmentCountRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[1], "", 1))
		}
	}

	if f.name == "" {
		if matches := somethingHashLikeRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[0], "", 1))
		}
	}

	if f.name == "" {
		if matches := likeFilenameRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.name = strings.TrimSpace(matches[1])
			subject = strings.TrimSpace(strings.Replace(subject, matches[1], "", 1))
		}
	}

	if p.fileCount > 0 && f.number == 0 {
		if matches := p.fileIndexRegex.FindStringSubmatch(subject); len(matches) == 2 {
			f.number = util.SafeParseInt(matches[1], 0)
			subject = strings.TrimSpace(strings.Replace(subject, matches[0], "", 1))
		}
	}

	if f.name == "" {
		f.name = subject
	}
}
