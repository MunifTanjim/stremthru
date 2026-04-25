package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

var magic = [8]byte{'P', 'A', 'R', '2', 0x00, 'P', 'K', 'T'}

var (
	typeMain      = [16]byte{'P', 'A', 'R', ' ', '2', '.', '0', 0x00, 'M', 'a', 'i', 'n', 0x00, 0x00, 0x00, 0x00}
	typeFileDesc  = [16]byte{'P', 'A', 'R', ' ', '2', '.', '0', 0x00, 'F', 'i', 'l', 'e', 'D', 'e', 's', 'c'}
	typeIFSC      = [16]byte{'P', 'A', 'R', ' ', '2', '.', '0', 0x00, 'I', 'F', 'S', 'C', 0x00, 0x00, 0x00, 0x00}
	typeRecvSlice = [16]byte{'P', 'A', 'R', ' ', '2', '.', '0', 0x00, 'R', 'e', 'c', 'v', 'S', 'l', 'i', 'c'}
	typeCreator   = [16]byte{'P', 'A', 'R', ' ', '2', '.', '0', 0x00, 'C', 'r', 'e', 'a', 't', 'o', 'r', 0x00}
)

const headerSize = 64
const maxPacketSize = 64 * 1024 * 1024

var (
	ErrInvalidMagic   = errors.New("par2: invalid magic bytes")
	ErrInvalidPacket  = errors.New("par2: invalid packet")
	ErrHashMismatch   = errors.New("par2: packet hash mismatch")
	ErrPacketTooLarge = errors.New("par2: packet too large")
)

type MainPacket struct {
	BlockSize       uint64
	RecoveryFileIDs [][16]byte
}

type FileDescriptionPacket struct {
	FileID   [16]byte
	MD5_16k  [16]byte
	MD5      [16]byte
	Length   uint64
	Filename string
}

type IFSCEntry struct {
	MD5   [16]byte
	CRC32 uint32
}

type IFSCPacket struct {
	FileID  [16]byte
	Entries []IFSCEntry
}

type RecoverySlicePacket struct {
	Exponent uint32
	Length   uint64
}

type File struct {
	Main           *MainPacket
	Files          []FileDescriptionPacket
	IFSCs          map[[16]byte]*IFSCPacket
	RecoverySlices []RecoverySlicePacket
	Creator        string
	RecoverySetID  [16]byte
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Decode() (*File, error) {
	f := &File{
		IFSCs: make(map[[16]byte]*IFSCPacket),
	}

	for {
		err := d.decodePacket(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (d *Decoder) decodePacket(f *File) error {
	var header [headerSize]byte
	if _, err := io.ReadFull(d.r, header[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("%w: truncated header", ErrInvalidPacket)
		}
		return err
	}

	if [8]byte(header[0:8]) != magic {
		return ErrInvalidMagic
	}

	packetLen := binary.LittleEndian.Uint64(header[8:16])
	if packetLen < headerSize {
		return fmt.Errorf("%w: packet length %d less than header size", ErrInvalidPacket, packetLen)
	}
	if packetLen > maxPacketSize {
		return fmt.Errorf("%w: packet length %d exceeds maximum %d", ErrPacketTooLarge, packetLen, maxPacketSize)
	}

	storedHash := [16]byte(header[16:32])

	if f.RecoverySetID == [16]byte{} {
		f.RecoverySetID = [16]byte(header[32:48])
	}

	packetType := [16]byte(header[48:64])

	bodyLen := packetLen - headerSize
	body := make([]byte, bodyLen)
	if bodyLen > 0 {
		if _, err := io.ReadFull(d.r, body); err != nil {
			if err == io.ErrUnexpectedEOF {
				return fmt.Errorf("%w: truncated body", ErrInvalidPacket)
			}
			return err
		}
	}

	h := md5.New()
	h.Write(header[32:64])
	h.Write(body)
	if [16]byte(h.Sum(nil)) != storedHash {
		return ErrHashMismatch
	}

	switch packetType {
	case typeMain:
		return parseMain(f, body)
	case typeFileDesc:
		return parseFileDesc(f, body)
	case typeIFSC:
		return parseIFSC(f, body)
	case typeRecvSlice:
		return parseRecvSlice(f, body, bodyLen)
	case typeCreator:
		return parseCreator(f, body)
	}

	return nil
}

func parseMain(f *File, body []byte) error {
	if len(body) < 8 {
		return fmt.Errorf("%w: main packet too short", ErrInvalidPacket)
	}

	f.Main = &MainPacket{
		BlockSize: binary.LittleEndian.Uint64(body[0:8]),
	}

	remaining := body[8:]
	count := len(remaining) / 16
	if count > 0 {
		f.Main.RecoveryFileIDs = make([][16]byte, count)
		for i := 0; i < count; i++ {
			f.Main.RecoveryFileIDs[i] = [16]byte(remaining[i*16 : (i+1)*16])
		}
	}

	return nil
}

func parseFileDesc(f *File, body []byte) error {
	// file_id(16) + md5(16) + md5_16k(16) + length(8) + filename(4+)
	if len(body) < 56 {
		return fmt.Errorf("%w: file description packet too short", ErrInvalidPacket)
	}

	fd := FileDescriptionPacket{
		FileID:  [16]byte(body[0:16]),
		MD5:     [16]byte(body[16:32]),
		MD5_16k: [16]byte(body[32:48]),
		Length:  binary.LittleEndian.Uint64(body[48:56]),
	}

	if len(body) > 56 {
		fd.Filename = strings.TrimRight(string(body[56:]), "\x00")
	}

	f.Files = append(f.Files, fd)
	return nil
}

func parseIFSC(f *File, body []byte) error {
	if len(body) < 16 {
		return fmt.Errorf("%w: IFSC packet too short", ErrInvalidPacket)
	}

	fileID := [16]byte(body[0:16])
	remaining := body[16:]
	count := len(remaining) / 20

	entries := make([]IFSCEntry, count)
	for i := 0; i < count; i++ {
		offset := i * 20
		entries[i] = IFSCEntry{
			MD5:   [16]byte(remaining[offset : offset+16]),
			CRC32: binary.LittleEndian.Uint32(remaining[offset+16 : offset+20]),
		}
	}

	f.IFSCs[fileID] = &IFSCPacket{
		FileID:  fileID,
		Entries: entries,
	}
	return nil
}

func parseRecvSlice(f *File, body []byte, bodyLen uint64) error {
	if len(body) < 4 {
		return fmt.Errorf("%w: recovery slice packet too short", ErrInvalidPacket)
	}

	f.RecoverySlices = append(f.RecoverySlices, RecoverySlicePacket{
		Exponent: binary.LittleEndian.Uint32(body[0:4]),
		Length:   bodyLen - 4,
	})
	return nil
}

func parseCreator(f *File, body []byte) error {
	f.Creator = string(bytes.TrimRight(body, "\x00"))
	return nil
}
