package usenet_pool

import (
	"regexp"
	"slices"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/nwaples/rardecode/v2"
)

type UsenetRARArchive struct {
	ufs      *UsenetFS
	Volumes  []UsenetRARVolume
	Password string
	solid    *bool
	files    []UsenetRARFile
}

func (ura *UsenetRARArchive) IsSolid() (bool, error) {
	if ura.solid == nil {
		opts := []rardecode.Option{rardecode.FileSystem(ura.ufs), rardecode.SkipCheck, rardecode.IterHeadersOnly, rardecode.IterSplitBlocks}
		if ura.Password != "" {
			opts = append(opts, rardecode.Password(ura.Password))
		}
		iter, err := rardecode.OpenIter(ura.Volumes[0].Filename, opts...)
		if err != nil {
			return false, err
		}
		defer iter.Close()

		solid := false
		for iter.Next() {
			if h := iter.Header(); h.Solid {
				solid = true
				break
			}
		}
		if err := iter.Err(); err != nil {
			return false, err
		}
		ura.solid = &solid
	}
	return *ura.solid, nil
}

func (ura *UsenetRARArchive) GetFiles() ([]UsenetRARFile, error) {
	if ura.files == nil {
		opts := []rardecode.Option{rardecode.FileSystem(ura.ufs), rardecode.SkipCheck, rardecode.IterHeadersOnly}
		if ura.Password != "" {
			opts = append(opts, rardecode.Password(ura.Password))
		}
		iter, err := rardecode.OpenIter(ura.Volumes[0].Filename, opts...)
		if err != nil {
			return nil, err
		}
		defer iter.Close()

		files := []UsenetRARFile{}
		for iter.Next() {
			header := iter.Header()
			files = append(files, UsenetRARFile{
				Name:         header.Name,
				PackedSize:   header.PackedSize,
				UnPackedSize: header.UnPackedSize,
				Solid:        header.Solid,
			})
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		ura.files = files
	}
	return ura.files, nil
}

func (ura *UsenetRARArchive) OpenFS() (*rardecode.RarFS, error) {
	opts := []rardecode.Option{rardecode.FileSystem(ura.ufs), rardecode.SkipCheck}
	if ura.Password != "" {
		opts = append(opts, rardecode.Password(ura.Password))
	}
	return rardecode.OpenFS(ura.Volumes[0].Filename, opts...)
}

type UsenetRARVolume struct {
	Number   int
	Filename string
	size     int64
	Segments []nzb.Segment
	Groups   []string
}

type UsenetRARFile struct {
	Name         string
	UnPackedSize int64
	PackedSize   int64
	Solid        bool
}

func (urf *UsenetRARFile) IsStreamable() bool {
	return !urf.Solid && urf.PackedSize == urf.UnPackedSize
}

// .part01.rar format
var rarPartNumberRegex = regexp.MustCompile(`(?i)\.part(\d+)\.rar$`)

// .r00, .r01 format (.rar is first part, .r00 is second, etc.)
var rarRNumberRegex = regexp.MustCompile(`(?i)\.r(\d+)$`)

// .rar
var rarFirstPartRegex = regexp.MustCompile(`(?i)\.rar$`)

func GetRARVolumeNumber(filename string) int {
	if matches := rarPartNumberRegex.FindStringSubmatch(filename); len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}

	if matches := rarRNumberRegex.FindStringSubmatch(filename); len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n + 1
	}

	if rarFirstPartRegex.MatchString(filename) {
		return 0
	}

	return -1
}

func NewUsenetRARArchive(ufs *UsenetFS) *UsenetRARArchive {
	volumes := []UsenetRARVolume{}
	for i := range ufs.nzb.Files {
		file := &ufs.nzb.Files[i]
		filename := file.GetName()
		volumeNumber := GetRARVolumeNumber(filename)
		if volumeNumber < 0 {
			continue
		}

		volumes = append(volumes, UsenetRARVolume{
			Number:   volumeNumber,
			Filename: filename,
			Segments: file.Segments,
			Groups:   file.Groups,
		})
	}
	slices.SortStableFunc(volumes, func(a, b UsenetRARVolume) int {
		return a.Number - b.Number
	})

	archive := &UsenetRARArchive{
		ufs:     ufs,
		Volumes: volumes,
	}

	return archive
}
