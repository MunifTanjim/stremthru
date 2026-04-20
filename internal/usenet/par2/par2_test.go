package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildPacket(recoverySetID [16]byte, packetType [16]byte, body []byte) []byte {
	packetLen := uint64(headerSize) + uint64(len(body))

	buf := make([]byte, packetLen)
	copy(buf[0:8], magic[:])
	binary.LittleEndian.PutUint64(buf[8:16], packetLen)
	// hash placeholder at [16:32]
	copy(buf[32:48], recoverySetID[:])
	copy(buf[48:64], packetType[:])
	copy(buf[64:], body)

	h := md5.Sum(buf[32:])
	copy(buf[16:32], h[:])

	return buf
}

var testRecoverySetID = [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

func TestDecodeMainPacket(t *testing.T) {
	fileID1 := [16]byte{0xAA, 0xBB, 0xCC, 0xDD}
	fileID2 := [16]byte{0x11, 0x22, 0x33, 0x44}

	body := make([]byte, 8+32)
	binary.LittleEndian.PutUint64(body[0:8], 65536)
	copy(body[8:24], fileID1[:])
	copy(body[24:40], fileID2[:])

	pkt := buildPacket(testRecoverySetID, typeMain, body)
	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	require.NotNil(t, f.Main)
	assert.Equal(t, uint64(65536), f.Main.BlockSize)
	assert.Equal(t, 2, len(f.Main.RecoveryFileIDs))
	assert.Equal(t, fileID1, f.Main.RecoveryFileIDs[0])
	assert.Equal(t, fileID2, f.Main.RecoveryFileIDs[1])
	assert.Equal(t, testRecoverySetID, f.RecoverySetID)
}

func TestDecodeFileDescPacket(t *testing.T) {
	fileID := [16]byte{0xAA}
	md5_16k := [16]byte{0xBB}
	md5Full := [16]byte{0xCC}
	filename := "test_movie.mkv"

	padded := filename
	for len(padded)%4 != 0 {
		padded += "\x00"
	}

	body := make([]byte, 56+len(padded))
	copy(body[0:16], fileID[:])
	copy(body[16:32], md5_16k[:])
	copy(body[32:48], md5Full[:])
	binary.LittleEndian.PutUint64(body[48:56], 1048576)
	copy(body[56:], padded)

	pkt := buildPacket(testRecoverySetID, typeFileDesc, body)
	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	require.Equal(t, 1, len(f.Files))
	assert.Equal(t, fileID, f.Files[0].FileID)
	assert.Equal(t, md5_16k, f.Files[0].MD5_16k)
	assert.Equal(t, md5Full, f.Files[0].MD5)
	assert.Equal(t, uint64(1048576), f.Files[0].Length)
	assert.Equal(t, filename, f.Files[0].Filename)
}

func TestDecodeIFSCPacket(t *testing.T) {
	fileID := [16]byte{0xAA}
	entry1MD5 := [16]byte{0x01, 0x02, 0x03}
	entry2MD5 := [16]byte{0x04, 0x05, 0x06}

	body := make([]byte, 16+40)
	copy(body[0:16], fileID[:])

	copy(body[16:32], entry1MD5[:])
	binary.LittleEndian.PutUint32(body[32:36], 0xDEADBEEF)

	copy(body[36:52], entry2MD5[:])
	binary.LittleEndian.PutUint32(body[52:56], 0xCAFEBABE)

	pkt := buildPacket(testRecoverySetID, typeIFSC, body)
	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	ifsc, ok := f.IFSCs[fileID]
	require.True(t, ok)
	assert.Equal(t, 2, len(ifsc.Entries))
	assert.Equal(t, entry1MD5, ifsc.Entries[0].MD5)
	assert.Equal(t, uint32(0xDEADBEEF), ifsc.Entries[0].CRC32)
	assert.Equal(t, entry2MD5, ifsc.Entries[1].MD5)
	assert.Equal(t, uint32(0xCAFEBABE), ifsc.Entries[1].CRC32)
}

func TestDecodeRecoverySlicePacket(t *testing.T) {
	recoveryData := make([]byte, 128)
	for i := range recoveryData {
		recoveryData[i] = byte(i)
	}

	body := make([]byte, 4+len(recoveryData))
	binary.LittleEndian.PutUint32(body[0:4], 7)
	copy(body[4:], recoveryData)

	pkt := buildPacket(testRecoverySetID, typeRecvSlice, body)
	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	require.Equal(t, 1, len(f.RecoverySlices))
	assert.Equal(t, uint32(7), f.RecoverySlices[0].Exponent)
	assert.Equal(t, uint64(128), f.RecoverySlices[0].Length)
}

func TestDecodeCreatorPacket(t *testing.T) {
	creator := "par2cmdline version 0.8.1"
	padded := creator + "\x00\x00\x00"

	pkt := buildPacket(testRecoverySetID, typeCreator, []byte(padded))
	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	assert.Equal(t, creator, f.Creator)
}

func TestDecodeMultiplePackets(t *testing.T) {
	fileID := [16]byte{0xAA}

	// Main packet
	mainBody := make([]byte, 8+16)
	binary.LittleEndian.PutUint64(mainBody[0:8], 32768)
	copy(mainBody[8:24], fileID[:])
	mainPkt := buildPacket(testRecoverySetID, typeMain, mainBody)

	// FileDesc packet
	filename := "video.mkv\x00\x00\x00" // pad to 12 bytes (multiple of 4)
	fdBody := make([]byte, 56+len(filename))
	copy(fdBody[0:16], fileID[:])
	binary.LittleEndian.PutUint64(fdBody[48:56], 999999)
	copy(fdBody[56:], filename)
	fdPkt := buildPacket(testRecoverySetID, typeFileDesc, fdBody)

	// Creator packet
	creatorPkt := buildPacket(testRecoverySetID, typeCreator, []byte("test\x00\x00\x00\x00"))

	var buf bytes.Buffer
	buf.Write(mainPkt)
	buf.Write(fdPkt)
	buf.Write(creatorPkt)

	dec := NewDecoder(&buf)
	f, err := dec.Decode()

	require.NoError(t, err)
	require.NotNil(t, f.Main)
	assert.Equal(t, uint64(32768), f.Main.BlockSize)
	assert.Equal(t, 1, len(f.Main.RecoveryFileIDs))
	require.Equal(t, 1, len(f.Files))
	assert.Equal(t, "video.mkv", f.Files[0].Filename)
	assert.Equal(t, uint64(999999), f.Files[0].Length)
	assert.Equal(t, "test", f.Creator)
}

func TestDecodeEmptyReader(t *testing.T) {
	dec := NewDecoder(bytes.NewReader(nil))
	f, err := dec.Decode()

	require.NoError(t, err)
	assert.Nil(t, f.Main)
	assert.Empty(t, f.Files)
}

func TestDecodeInvalidMagic(t *testing.T) {
	data := make([]byte, headerSize)
	copy(data[0:8], []byte("NOTPAR2!"))

	dec := NewDecoder(bytes.NewReader(data))
	_, err := dec.Decode()

	assert.ErrorIs(t, err, ErrInvalidMagic)
}

func TestDecodeTruncatedHeader(t *testing.T) {
	dec := NewDecoder(bytes.NewReader(magic[:]))
	_, err := dec.Decode()

	assert.ErrorIs(t, err, ErrInvalidPacket)
}

func TestDecodeHashMismatch(t *testing.T) {
	pkt := buildPacket(testRecoverySetID, typeCreator, []byte("hello\x00\x00\x00"))
	// corrupt the body to cause hash mismatch
	pkt[len(pkt)-1] = 0xFF

	dec := NewDecoder(bytes.NewReader(pkt))
	_, err := dec.Decode()

	assert.ErrorIs(t, err, ErrHashMismatch)
}

func TestDecodeUnknownPacketType(t *testing.T) {
	unknownType := [16]byte{0xFF, 0xFF}
	pkt := buildPacket(testRecoverySetID, unknownType, []byte("data"))

	dec := NewDecoder(bytes.NewReader(pkt))
	f, err := dec.Decode()

	require.NoError(t, err)
	assert.Nil(t, f.Main)
	assert.Empty(t, f.Files)
}

func TestDecodePacketTooShort(t *testing.T) {
	pkt := buildPacket(testRecoverySetID, typeMain, []byte{0x01, 0x02})

	dec := NewDecoder(bytes.NewReader(pkt))
	_, err := dec.Decode()

	assert.ErrorIs(t, err, ErrInvalidPacket)
}
