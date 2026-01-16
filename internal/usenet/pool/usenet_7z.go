package usenet_pool

import (
	"errors"
	"io"
	"regexp"
	"slices"
	"strconv"

	"github.com/bodgit/sevenzip"
	"github.com/spf13/afero"
)

var (
	_ Archive     = (*SevenZipArchive)(nil)
	_ ArchiveFile = (*Usenet7zFile)(nil)
)

type SevenZipArchive struct {
	fs    afero.Fs
	name  string
	r     *sevenzip.ReadCloser
	files []ArchiveFile
}

func (usa *SevenZipArchive) Open(password string) error {
	opts := []sevenzip.ReaderOption{sevenzip.WithFs(usa.fs)}
	if password != "" {
		opts = append(opts, sevenzip.WithPassword(password))
	}
	reader, err := sevenzip.OpenReader(usa.name, opts...)
	if err != nil {
		return err
	}
	usa.r = reader
	return nil
}

func (usa *SevenZipArchive) Close() error {
	var errs []error
	if usa.r != nil {
		if err := usa.r.Close(); err != nil {
			errs = append(errs, err)
		}
		usa.r = nil
	}
	if c, ok := usa.fs.(io.Closer); ok {
		if err := c.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (usa *SevenZipArchive) GetFiles() ([]ArchiveFile, error) {
	if usa.files == nil {
		iter := usa.r.Iter()
		files := []ArchiveFile{}
		for iter.Next() {
			entry := iter.Entry()
			file := &Usenet7zFile{
				ArchiveEntry: entry,
				name:         entry.Name(),
				unPackedSize: entry.Size(),
				packedSize:   entry.CompressedSize,
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

func (usa *SevenZipArchive) IsStreamable() bool {
	iter := usa.r.Iter()
	return !iter.HasCompression && !iter.HasEncryption
}

type Usenet7zFile struct {
	*sevenzip.ArchiveEntry
	name         string
	unPackedSize int64
	packedSize   int64
}

func (f *Usenet7zFile) Name() string {
	return f.name
}

func (f *Usenet7zFile) UnPackedSize() int64 {
	return f.unPackedSize
}

func (f *Usenet7zFile) PackedSize() int64 {
	return f.packedSize
}

func (f *Usenet7zFile) IsStreamable() bool {
	return !f.ArchiveEntry.IsCompressed() && !f.ArchiveEntry.IsEncrypted()
}

func (f *Usenet7zFile) Open() (io.ReadSeekCloser, error) {
	r, err := f.ArchiveEntry.Open()
	if err != nil {
		return nil, err
	}
	return r.(io.ReadSeekCloser), nil
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

func NewUsenetSevenZipArchive(ufs *UsenetFS) *SevenZipArchive {
	volumes := []archiveVolume{}
	for i := range ufs.nzb.Files {
		file := &ufs.nzb.Files[i]
		name := file.GetName()
		n := Get7zVolumeNumber(name)
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

	return &SevenZipArchive{
		fs:   ufs.toAfero(),
		name: firstVolume,
	}
}

func NewSevenZipArchive(fs afero.Fs, name string) *SevenZipArchive {
	return &SevenZipArchive{fs: fs, name: name}
}
