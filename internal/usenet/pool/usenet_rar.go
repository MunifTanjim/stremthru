package usenet_pool

import (
	"io"
	"io/fs"
	"regexp"
	"slices"
	"strconv"

	"github.com/nwaples/rardecode/v2"
)

var (
	_ Archive     = (*RARArchive)(nil)
	_ ArchiveFile = (*UsenetRARFile)(nil)
)

type RARArchive struct {
	fs       fs.FS
	name     string
	solid    *bool
	files    []ArchiveFile
	password string
	r        *rardecode.RarFS
}

func (ura *RARArchive) Open(password string) error {
	ura.password = password

	opts := []rardecode.Option{rardecode.FileSystem(ura.fs), rardecode.SkipCheck}
	if password != "" {
		opts = append(opts, rardecode.Password(password))
	}
	r, err := rardecode.OpenFS(ura.name, opts...)
	if err != nil {
		return err
	}
	ura.r = r
	return nil
}

func (ura *RARArchive) Close() error {
	if c, ok := ura.fs.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (ura *RARArchive) IsStreamable() bool {
	solid, err := ura.isSolid()
	return err == nil && !solid
}

func (ura *RARArchive) isSolid() (bool, error) {
	if ura.solid == nil {
		opts := []rardecode.Option{rardecode.FileSystem(ura.fs), rardecode.SkipCheck, rardecode.IterHeadersOnly, rardecode.IterSplitBlocks}
		if ura.password != "" {
			opts = append(opts, rardecode.Password(ura.password))
		}
		iter, err := rardecode.OpenIter(ura.name, opts...)
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

func (ura *RARArchive) GetFiles() ([]ArchiveFile, error) {
	if ura.files == nil {
		opts := []rardecode.Option{rardecode.FileSystem(ura.fs), rardecode.SkipCheck, rardecode.IterHeadersOnly}
		if ura.password != "" {
			opts = append(opts, rardecode.Password(ura.password))
		}
		iter, err := rardecode.OpenIter(ura.name, opts...)
		if err != nil {
			return nil, err
		}
		defer iter.Close()

		files := []ArchiveFile{}
		for iter.Next() {
			header := iter.Header()
			file := &UsenetRARFile{
				r:            ura.r,
				name:         header.Name,
				packedSize:   header.PackedSize,
				unPackedSize: header.UnPackedSize,
				solid:        header.Solid,
			}
			files = append(files, file)
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
		ura.files = files
	}
	return ura.files, nil
}

type UsenetRARFile struct {
	r            *rardecode.RarFS
	name         string
	unPackedSize int64
	packedSize   int64
	solid        bool
}

func (urf *UsenetRARFile) Name() string {
	return urf.name
}

func (urf *UsenetRARFile) Open() (io.ReadSeekCloser, error) {
	r, err := urf.r.Open(urf.name)
	if err != nil {
		return nil, err
	}
	return r.(io.ReadSeekCloser), nil
}

func (urf *UsenetRARFile) PackedSize() int64 {
	return urf.packedSize
}

func (urf *UsenetRARFile) UnPackedSize() int64 {
	return urf.unPackedSize
}

func (urf *UsenetRARFile) IsStreamable() bool {
	return !urf.solid && urf.packedSize == urf.unPackedSize
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

func NewUsenetRARArchive(ufs *UsenetFS) *RARArchive {
	volumes := []archiveVolume{}
	for i := range ufs.nzb.Files {
		file := &ufs.nzb.Files[i]
		name := file.GetName()
		n := GetRARVolumeNumber(name)
		if n < 0 {
			continue
		}
		volumes = append(volumes, archiveVolume{n: n, name: name})
	}
	slices.SortStableFunc(volumes, func(a, b archiveVolume) int {
		return a.n - b.n
	})

	var firstVolume string
	if len(volumes) > 0 {
		firstVolume = volumes[0].name
	}

	return &RARArchive{
		fs:   ufs,
		name: firstVolume,
	}
}

func NewRARArchive(fs fs.FS, name string) *RARArchive {
	return &RARArchive{fs: fs, name: name}
}
