package usenet_pool

import (
	"io"
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectFileStreaming(t *testing.T) {
	t.Run("YEncDecodingThroughPipeline", func(t *testing.T) {
		originalData := []byte("Hello, this is test data for yEnc streaming!")
		encoded := encodeYenc(originalData, "test.bin", 1, 1, int64(len(originalData)), 1)

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.test", "211 1 1 1 alt.test")
		server.SetResponse(
			"BODY <test@example.com>",
			"222 0 <test@example.com>",
			strings.Split(strings.TrimSpace(string(encoded)), "\r\n"),
		)
		server.Start(t)

		nntpPool := nntptest.NewPool(t, server, &nntp.PoolConfig{})

		usenetPool := &Pool{
			Log:          logger.Scoped("test/usenet/pool"),
			providers:    []*providerPool{{Pool: nntpPool}},
			segmentCache: NewSegmentCache(10 * 1024 * 1024),
		}

		segments := []nzb.Segment{
			{MessageId: "test@example.com", Bytes: int64(len(encoded)), Number: 1},
		}

		ctx := t.Context()
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments:   segments,
			Groups:     []string{"alt.test"},
			BufferSize: 1,
		})
		require.NoError(t, err)
		defer result.Close()

		decoded, err := io.ReadAll(result)
		require.NoError(t, err)

		assert.Equal(t, originalData, decoded)
	})

	t.Run("MultiSegmentSequentialRead", func(t *testing.T) {
		segment1Data := []byte("First segment data - part 1")
		segment2Data := []byte("Second segment data - part 2")
		segment3Data := []byte("Third segment data - part 3")
		totalSize := int64(len(segment1Data) + len(segment2Data) + len(segment3Data))

		encoded1 := encodeYenc(segment1Data, "test.bin", 1, 3, totalSize, 1)
		encoded2 := encodeYenc(segment2Data, "test.bin", 2, 3, totalSize, int64(len(segment1Data))+1)
		encoded3 := encodeYenc(segment3Data, "test.bin", 3, 3, totalSize, int64(len(segment1Data)+len(segment2Data))+1)

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.test", "211 3 1 3 alt.test")

		msgIds := []string{"seg1@test.com", "seg2@test.com", "seg3@test.com"}
		for i, encoded := range [][]byte{encoded1, encoded2, encoded3} {
			msgId := msgIds[i]
			lines := strings.Split(strings.TrimSpace(string(encoded)), "\r\n")
			server.SetResponse("BODY <"+msgId+">", "222 0 <"+msgId+">", lines)
		}
		server.Start(t)

		nntpPool := nntptest.NewPool(t, server, &nntp.PoolConfig{})

		usenetPool := &Pool{
			Log:          logger.Scoped("test/usenet/pool"),
			providers:    []*providerPool{{Pool: nntpPool}},
			segmentCache: NewSegmentCache(10 * 1024 * 1024),
		}

		segments := []nzb.Segment{
			{MessageId: msgIds[0], Bytes: int64(len(encoded1)), Number: 1},
			{MessageId: msgIds[1], Bytes: int64(len(encoded2)), Number: 2},
			{MessageId: msgIds[2], Bytes: int64(len(encoded3)), Number: 3},
		}

		ctx := t.Context()
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments:   segments,
			Groups:     []string{"alt.test"},
			BufferSize: 2,
		})
		require.NoError(t, err)
		defer result.Close()

		decoded, err := io.ReadAll(result)
		require.NoError(t, err)

		expected := append(append(segment1Data, segment2Data...), segment3Data...)
		assert.Equal(t, expected, decoded)
	})

	t.Run("SegmentsStreamPartialReads", func(t *testing.T) {
		originalData := makeTestBytes(500)
		encoded := encodeYenc(originalData, "test.bin", 1, 1, int64(len(originalData)), 1)

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.test", "211 1 1 1 alt.test")
		lines := strings.Split(strings.TrimSpace(string(encoded)), "\r\n")
		server.SetResponse("BODY <partial@test.com>", "222 0 <partial@test.com>", lines)
		server.Start(t)

		nntpPool := nntptest.NewPool(t, server, &nntp.PoolConfig{})

		usenetPool := &Pool{
			Log:          logger.Scoped("test/usenet/pool"),
			providers:    []*providerPool{{Pool: nntpPool}},
			segmentCache: NewSegmentCache(10 * 1024 * 1024),
		}

		segments := []nzb.Segment{
			{MessageId: "partial@test.com", Bytes: int64(len(encoded)), Number: 1},
		}

		ctx := t.Context()
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments:   segments,
			Groups:     []string{"alt.test"},
			BufferSize: 1,
		})
		require.NoError(t, err)
		defer result.Close()

		var decoded []byte
		buf := make([]byte, 17)
		for {
			n, err := result.Read(buf)
			if n > 0 {
				decoded = append(decoded, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		assert.Equal(t, originalData, decoded)
	})

	t.Run("EmptySegmentsError", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.Start(t)

		nntpPool := nntptest.NewPool(t, server, &nntp.PoolConfig{})

		usenetPool := &Pool{
			Log:          logger.Scoped("test/usenet/pool"),
			providers:    []*providerPool{{Pool: nntpPool}},
			segmentCache: NewSegmentCache(10 * 1024 * 1024),
		}

		ctx := t.Context()
		_, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: []nzb.Segment{},
			Groups:   []string{"alt.test"},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no segments")
	})
}

func TestStreamSegmentsAPI(t *testing.T) {
	t.Run("SizeFromYEncHeader", func(t *testing.T) {
		totalFileSize := int64(3500)
		segment1Data := makeTestBytes(1000)
		encoded1 := encodeYenc(segment1Data, "test.bin", 1, 3, totalFileSize, 1)

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.test", "211 3 1 3 alt.test")
		lines := strings.Split(strings.TrimSpace(string(encoded1)), "\r\n")
		server.SetResponse("BODY <seg1@test.com>", "222 0 <seg1@test.com>", lines)
		server.Start(t)

		nntpPool := nntptest.NewPool(t, server, &nntp.PoolConfig{})

		usenetPool := &Pool{
			Log:          logger.Scoped("test/usenet/pool"),
			providers:    []*providerPool{{Pool: nntpPool}},
			segmentCache: NewSegmentCache(10 * 1024 * 1024),
		}

		segments := []nzb.Segment{
			{MessageId: "seg1@test.com", Bytes: int64(len(encoded1)), Number: 1},
		}

		ctx := t.Context()
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: segments,
			Groups:   []string{"alt.test"},
		})
		require.NoError(t, err)
		defer result.Close()

		assert.Equal(t, totalFileSize, result.Size)
	})
}
