package nzb

import (
	"bytes"
	"encoding/xml"
	"io"
	"slices"
	"strings"

	"golang.org/x/net/html/charset"
)

type ParseError struct {
	Message string
	Cause   error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

type Meta struct {
	XMLName xml.Name `xml:"meta"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",chardata"`
}

type Head struct {
	XMLName xml.Name `xml:"head"`
	Meta    []Meta   `xml:"meta"`
}

type Segment struct {
	XMLName   xml.Name `xml:"segment"`
	Bytes     int64    `xml:"bytes,attr"`
	Number    int      `xml:"number,attr"`
	MessageId string   `xml:",chardata"`
}

type File struct {
	XMLName  xml.Name  `xml:"file"`
	Poster   string    `xml:"poster,attr"`
	Date     int64     `xml:"date,attr"` // unix second
	Subject  string    `xml:"subject,attr"`
	Groups   []string  `xml:"groups>group"`
	Segments []Segment `xml:"segments>segment"`

	name       string   `xml:"-"`
	number     int      `xml:"-"`
	totalSize  int64    `xml:"-"`
	messageIds []string `xml:"-"`
}

func (f *File) Name() string {
	return f.name
}

type NZB struct {
	XMLName xml.Name `xml:"nzb"`
	Head    *Head    `xml:"head"`
	Files   []File   `xml:"file"`

	subjectParsed bool `xml:"-"`
}

func (n *NZB) ParseFileSubject() {
	if n.subjectParsed {
		return
	}
	n.subjectParsed = true
	subjectParser := newSubjectParser(len(n.Files))
	for i := range n.Files {
		f := &n.Files[i]
		subjectParser.Parse(f)
	}
}

func Parse(r io.Reader) (*NZB, error) {
	decoder := xml.NewDecoder(r)
	decoder.CharsetReader = charset.NewReaderLabel

	var nzb NZB
	if err := decoder.Decode(&nzb); err != nil {
		return nil, &ParseError{
			Message: "Failed to parse",
			Cause:   err,
		}
	}

	nzb.ParseFileSubject()

	slices.SortStableFunc(nzb.Files, func(a, b File) int {
		return a.number - b.number
	})

	for i := range nzb.Files {
		f := &nzb.Files[i]
		slices.SortStableFunc(f.Segments, func(a, b Segment) int {
			return a.Number - b.Number
		})
	}

	return &nzb, nil
}

func ParseBytes(data []byte) (*NZB, error) {
	return Parse(bytes.NewReader(data))
}

func (n *NZB) TotalSize() (bytes int64) {
	for i := range n.Files {
		bytes += n.Files[i].Size()
	}
	return bytes
}

func (n *NZB) FileCount() int {
	return len(n.Files)
}

func (n *NZB) GetMeta(metaType string) string {
	if n.Head == nil {
		return ""
	}
	for _, m := range n.Head.Meta {
		if m.Type == metaType {
			return m.Value
		}
	}
	return ""
}

func (n *NZB) GetLargestFileIdx(skip func(filename string) bool) int {
	largestIdx := -1
	largestSize := int64(0)
	for i := range n.Files {
		if skip != nil && skip(n.Files[i].Name()) {
			continue
		}
		size := n.Files[i].Size()
		if size > largestSize {
			largestSize = size
			largestIdx = i
		}
	}
	return largestIdx
}

func (f *File) Size() int64 {
	if f.totalSize == 0 {
		var bytes int64
		for i := range f.Segments {
			bytes += f.Segments[i].Bytes
		}
		f.totalSize = bytes
	}
	return f.totalSize
}

func (f *File) MessageIds() []string {
	if f.messageIds == nil {
		ids := make([]string, len(f.Segments))
		for i := range f.Segments {
			ids[i] = strings.TrimSpace(f.Segments[i].MessageId)
		}
		f.messageIds = ids
	}
	return f.messageIds
}

func (f *File) SegmentCount() int {
	return len(f.Segments)
}
