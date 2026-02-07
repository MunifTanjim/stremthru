package usenet_pool

import (
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestNZB(files ...nzb.File) *nzb.NZB {
	if len(files) == 0 {
		return &nzb.NZB{Files: []nzb.File{}}
	}

	nzbDoc := &nzb.NZB{
		Files: make([]nzb.File, len(files)),
	}

	for i, f := range files {
		if len(f.Groups) == 0 {
			f.Groups = []string{"alt.binaries.test"}
		}
		if f.Date == 0 {
			f.Date = time.Now().Unix()
		}
		if f.Poster == "" {
			f.Poster = "test@example.com"
		}

		nzbDoc.Files[i] = f
	}

	nzbDoc.ParseFileSubject()

	return nzbDoc
}

func createTestPool(t *testing.T, server *nntptest.Server) *Pool {
	t.Helper()

	pool, err := NewPool(&Config{
		Providers: []ProviderConfig{
			{
				PoolConfig: nntp.PoolConfig{
					ConnectionConfig: nntp.ConnectionConfig{
						Host: server.Host(),
						Port: server.Port(),
					},
				},
			},
		},
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestUsenetFS_Open(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "existing.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB: nzbDoc,
		})

		f, err := ufs.Open("nonexistent.txt")
		assert.Nil(t, f)
		assert.Error(t, err)

		var pathErr *fs.PathError
		require.ErrorAs(t, err, &pathErr)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "nonexistent.txt", pathErr.Path)
		assert.ErrorIs(t, pathErr.Err, fs.ErrNotExist)
	})

	t.Run("EmptyName", func(t *testing.T) {
		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "file.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB: nzbDoc,
		})

		f, err := ufs.Open("")
		assert.Nil(t, f)
		assert.Error(t, err)

		var pathErr *fs.PathError
		require.ErrorAs(t, err, &pathErr)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, ".", pathErr.Path)
		assert.ErrorIs(t, pathErr.Err, fs.ErrNotExist)
	})

	t.Run("ValidFile", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		pool := createTestPool(t, server)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		require.NotNil(t, f)
		f.Close()

		uf, ok := f.(*UsenetFile)
		assert.True(t, ok)
		assert.NotNil(t, uf)
	})

	t.Run("PathCleaning", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "file.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		pool := createTestPool(t, server)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		testCases := []struct {
			path string
		}{
			{"./file.txt"},
			{"dir/../file.txt"},
			{"./dir/../file.txt"},
		}

		for _, tc := range testCases {
			t.Run(tc.path, func(t *testing.T) {
				f, err := ufs.Open(tc.path)
				require.NoError(t, err, "path: %s", tc.path)
				require.NotNil(t, f)
				f.Close()
			})
		}
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencEncoded)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(
			nzb.File{
				Subject: `Test - "file1.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg1@test", Bytes: 100, Number: 1},
				},
			},
			nzb.File{
				Subject: `Test - "file2.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg2@test", Bytes: 200, Number: 1},
				},
			},
			nzb.File{
				Subject: `Test - "file3.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg3@test", Bytes: 300, Number: 1},
				},
			},
		)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		for _, name := range []string{"file1.bin", "file2.bin", "file3.bin"} {
			f, err := ufs.Open(name)
			require.NoError(t, err, "file: %s", name)
			require.NotNil(t, f)
			f.Close()
		}
	})
}

func TestUsenetFile_Stat(t *testing.T) {
	t.Run("ReturnsFileInfo", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 2, 3000, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/2)`,
			Date:    1234567890,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 1000, Number: 1},
				{MessageId: "msg2@test", Bytes: 2000, Number: 2},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		fi, err := f.Stat()
		require.NoError(t, err)
		assert.Equal(t, "test.bin", fi.Name())
		assert.Equal(t, int64(3000), fi.Size())
		assert.Equal(t, fs.FileMode(0644), fi.Mode())
		assert.Equal(t, time.Unix(1234567890, 0), fi.ModTime())
		assert.False(t, fi.IsDir())
		assert.Nil(t, fi.Sys())
	})
}

func TestUsenetFile_Read(t *testing.T) {
	t.Run("SingleSegment", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		buf := make([]byte, 200)
		n, err := f.Read(buf)
		assert.Equal(t, 100, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData, buf[:n])

		n, err = f.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("MultipleReads", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		// Read in small chunks
		buf := make([]byte, 30)
		var result []byte

		for {
			n, err := f.Read(buf)
			if n > 0 {
				result = append(result, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		assert.Equal(t, originalData, result)
	})
}

func TestUsenetFile_Seek(t *testing.T) {
	setupSeekTest := func(t *testing.T, dataSize int) (*UsenetFile, []byte, func()) {
		t.Helper()

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		originalData := makeTestBytes(dataSize)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, int64(dataSize), 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: int64(dataSize), Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		cleanup := func() {
			f.Close()
		}

		return uf, originalData, cleanup
	}

	t.Run("SeekStart", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Seek to position 50
		pos, err := uf.Seek(50, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(50), pos)

		// Read from that position
		buf := make([]byte, 10)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[50:60], buf[:n])
	})

	t.Run("SeekCurrent", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Read 20 bytes to position at 20
		buf := make([]byte, 20)
		_, err := uf.Read(buf)
		require.NoError(t, err)

		// Seek forward 10 from current (20 + 10 = 30)
		pos, err := uf.Seek(10, io.SeekCurrent)
		require.NoError(t, err)
		assert.Equal(t, int64(30), pos)

		// Read from that position
		n, err := uf.Read(buf[:10])
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[30:40], buf[:n])
	})

	t.Run("SeekEnd", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Seek to 10 bytes before end
		pos, err := uf.Seek(-10, io.SeekEnd)
		require.NoError(t, err)
		assert.Equal(t, int64(90), pos)

		// Read from that position
		buf := make([]byte, 20)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[90:], buf[:n])
	})

	t.Run("SeekThenRead", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 200)
		defer cleanup()

		// Test multiple seek and read operations
		// Each test seeks to a position and reads 10 bytes
		testCases := []struct {
			offset int64
			whence int
			want   int64
		}{
			{0, io.SeekStart, 0},
			{50, io.SeekStart, 50},
			{100, io.SeekStart, 100},
			{-50, io.SeekEnd, 150},
		}

		for _, tc := range testCases {
			pos, err := uf.Seek(tc.offset, tc.whence)
			require.NoError(t, err)
			assert.Equal(t, tc.want, pos)

			// Verify read returns correct data
			buf := make([]byte, 10)
			n, err := uf.Read(buf)
			assert.Equal(t, 10, n)
			assert.NoError(t, err)
			assert.Equal(t, originalData[tc.want:tc.want+10], buf[:n])
		}
	})

	t.Run("SeekNegativePosition", func(t *testing.T) {
		uf, _, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		_, err := uf.Seek(-10, io.SeekStart)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "negative position")
	})

	t.Run("SeekSkipsIntermediateSegments", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		segment1Data := makeTestBytes(100)
		segment2Data := makeTestBytes(100)
		segment3Data := makeTestBytes(100)
		segment4Data := makeTestBytes(100)

		yencSeg1 := encodeYenc(segment1Data, "test.bin", 1, 4, 400, 1)
		yencSeg2 := encodeYenc(segment2Data, "test.bin", 2, 4, 400, 101)
		yencSeg3 := encodeYenc(segment3Data, "test.bin", 3, 4, 400, 201)
		yencSeg4 := encodeYenc(segment4Data, "test.bin", 4, 4, 400, 301)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencSeg1)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencSeg2)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencSeg3)})
		server.SetResponse("BODY <msg4@test>", "222 0 <msg4@test>", []string{string(yencSeg4)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/4)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
				{MessageId: "msg2@test", Bytes: 100, Number: 2},
				{MessageId: "msg3@test", Bytes: 100, Number: 3},
				{MessageId: "msg4@test", Bytes: 100, Number: 4},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:               nzbDoc,
			Pool:              pool,
			SegmentBufferSize: 1,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		// Step 1: Read 10 bytes from start (should fetch segment 1)
		buf := make([]byte, 10)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, segment1Data[:10], buf[:n])

		// Step 2: Seek to position 350 (inside segment 4, skipping segment 2,3)
		pos, err := uf.Seek(350, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(350), pos)

		// Step 3: Read 10 bytes from position 350 (should fetch segment 4)
		n, err = uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		// Position 350 is offset 50 in segment 4 (350 - 300 = 50)
		assert.Equal(t, segment4Data[50:60], buf[:n])

		// Assert segment 1, 2 (buffered) and 4 were fetched, but not segment 3
		cmds := server.GetRequestCommands()
		assert.True(t, cmds.HasCommand("BODY <msg1@test>"), "segment 1 should be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg2@test>"), "segment 2 should not be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg3@test>"), "segment 3 should not be fetched")
		assert.True(t, cmds.HasCommand("BODY <msg4@test>"), "segment 4 should be fetched")
	})
}

func TestUsenetFile_ReadAt(t *testing.T) {
	setup := func(t *testing.T) (*UsenetFile, []byte, *nntptest.Server) {
		t.Helper()

		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		// Create 4 segments of 50 bytes each (total: 200 bytes)
		segment1Data := makeTestBytes(50)
		segment2Data := makeTestBytes(50)
		segment3Data := makeTestBytes(50)
		segment4Data := makeTestBytes(50)

		// Combine all segments to create the complete file data
		data := make([]byte, 0, 200)
		data = append(data, segment1Data...)
		data = append(data, segment2Data...)
		data = append(data, segment3Data...)
		data = append(data, segment4Data...)

		// Encode each segment as yEnc
		yencSeg1 := encodeYenc(segment1Data, "test.bin", 1, 4, 200, 1)
		yencSeg2 := encodeYenc(segment2Data, "test.bin", 2, 4, 200, 51)
		yencSeg3 := encodeYenc(segment3Data, "test.bin", 3, 4, 200, 101)
		yencSeg4 := encodeYenc(segment4Data, "test.bin", 4, 4, 200, 151)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencSeg1)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencSeg2)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencSeg3)})
		server.SetResponse("BODY <msg4@test>", "222 0 <msg4@test>", []string{string(yencSeg4)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/4)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 50, Number: 1},
				{MessageId: "msg2@test", Bytes: 50, Number: 2},
				{MessageId: "msg3@test", Bytes: 50, Number: 3},
				{MessageId: "msg4@test", Bytes: 50, Number: 4},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		t.Cleanup(func() {
			f.Close()
		})

		return uf, data, server
	}

	t.Run("ReadAtOffset", func(t *testing.T) {
		uf, data, _ := setup(t)

		buf := make([]byte, 20)
		n, err := uf.ReadAt(buf, 50)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[50:70], buf[:n])

		n, err = uf.ReadAt(buf, 100)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[100:120], buf[:n])
	})

	t.Run("ReadAtDoesNotAffectPosition", func(t *testing.T) {
		uf, data, _ := setup(t)

		buf := make([]byte, 20)
		n, err := uf.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 20, n)
		assert.Equal(t, data[0:20], buf[:n])

		n, err = uf.ReadAt(buf, 100)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[100:120], buf[:n])

		n, err = uf.Read(buf)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[20:40], buf[:n])
	})

	t.Run("ReadAtSkipsIntermediateSegments", func(t *testing.T) {
		uf, data, server := setup(t)

		buf := make([]byte, 10)
		n, err := uf.ReadAt(buf, 160)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, data[160:170], buf[:n])

		cmds := server.GetRequestCommands()
		assert.True(t, cmds.HasCommand("BODY <msg1@test>"), "segment 1 should be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg2@test>"), "segment 2 should NOT be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg3@test>"), "segment 3 should NOT be fetched")
		assert.True(t, cmds.HasCommand("BODY <msg4@test>"), "segment 4 should be fetched")
	})
}
