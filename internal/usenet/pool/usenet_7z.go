package usenet_pool

import (
	"io"
	"regexp"
	"slices"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/bodgit/sevenzip"
)

type Usenet7zArchive struct {
	ufs     *UsenetFS
	r       *sevenzip.ReadCloser
	Volumes []Usenet7zVolume
	files   []Usenet7zFile
}

func (usa *Usenet7zArchive) Open(password string) error {
	opts := []sevenzip.ReaderOption{sevenzip.WithFs(usa.ufs.toAfero())}
	if password != "" {
		opts = append(opts, sevenzip.WithPassword(password))
	}
	reader, err := sevenzip.OpenReader(usa.Volumes[0].Filename, opts...)
	if err != nil {
		return err
	}
	usa.r = reader
	return nil
}

func (usa *Usenet7zArchive) Close() error {
	if usa.r != nil {
		err := usa.r.Close()
		usa.r = nil
		return err
	}
	return nil
}

func (usa *Usenet7zArchive) GetFiles() ([]Usenet7zFile, error) {
	if usa.files == nil {
		iter := usa.r.Iter()
		files := []Usenet7zFile{}
		for iter.Next() {
			entry := iter.Entry()
			println(entry.Name())
			file := Usenet7zFile{
				ArchiveEntry: entry,
				Name:         entry.Name(),
				UnPackedSize: entry.Size(),
				PackedSize:   entry.CompressedSize,
			}
			files = append(files, file)
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		usa.files = files
	}
	return usa.files, nil
}

type Usenet7zVolume struct {
	Number   int
	Filename string
	size     int64
	Segments []nzb.Segment
	Groups   []string
}

type Usenet7zFile struct {
	*sevenzip.ArchiveEntry
	Name         string
	UnPackedSize int64
	PackedSize   int64
}

func (usf *Usenet7zFile) toStream() (*Stream, error) {
	r, err := usf.Open()
	if err != nil {
		return nil, err
	}
	return &Stream{
		ReadSeekCloser: r.(io.ReadSeekCloser),
		Name:           usf.Name,
		Size:           usf.UnPackedSize,
		ContentType:    GetContentType(usf.Name),
	}, nil
}

// .7z.001, .7z.002 format
var sevenzipPartNumberRegex = regexp.MustCompile(`(?i)\.7z\.(\d+)$`)

// .7z
var sevenzipFirstPartRegex = regexp.MustCompile(`(?i)\.7z$`)

func Get7zVolumeNumber(filename string) int {
	if matches := sevenzipPartNumberRegex.FindStringSubmatch(filename); len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}

	if sevenzipFirstPartRegex.MatchString(filename) {
		return 0
	}

	return -1
}

func NewUsenet7zArchive(ufs *UsenetFS) *Usenet7zArchive {
	volumes := []Usenet7zVolume{}

	for i := range ufs.nzb.Files {
		file := &ufs.nzb.Files[i]
		filename := file.GetName()
		volumeNumber := Get7zVolumeNumber(filename)
		if volumeNumber < 0 {
			continue
		}

		volumes = append(volumes, Usenet7zVolume{
			Number:   volumeNumber,
			Filename: filename,
			Segments: file.Segments,
			Groups:   file.Groups,
		})
	}
	slices.SortStableFunc(volumes, func(a, b Usenet7zVolume) int {
		return a.Number - b.Number
	})

	archive := &Usenet7zArchive{
		ufs:     ufs,
		Volumes: volumes,
	}

	return archive
}
