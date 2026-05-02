package usenet_pool

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSegmentCache implements SegmentCache for testing
type mockSegmentCache struct {
	mu   sync.RWMutex
	data map[string]SegmentData
}

func newMockSegmentCache() *mockSegmentCache {
	return &mockSegmentCache{
		data: make(map[string]SegmentData),
	}
}

func (m *mockSegmentCache) Get(messageId string) (SegmentData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[messageId]
	return data, ok
}

func (m *mockSegmentCache) Set(messageId string, data SegmentData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[messageId] = data
}

func (m *mockSegmentCache) Prepopulate(messageId string, data SegmentData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[messageId] = data
}

// Helper functions for testing

func setupServerWithSegment(t *testing.T, server *nntptest.Server, messageId string, data []byte) {
	t.Helper()
	encoded := encodeYenc(data, "test.bin", 1, 1, int64(len(data)), 1)
	lines := strings.Split(strings.TrimSpace(string(encoded)), "\r\n")
	server.SetResponse("BODY <"+messageId+">", "222 0 <"+messageId+">", lines)
}

func setupServerConnectionError(server *nntptest.Server, messageId string) {
	server.SetResponse("BODY <"+messageId+">", "400 Service temporarily unavailable")
}

func setupServerArticleNotFound(server *nntptest.Server, messageId string) {
	server.SetResponse("BODY <"+messageId+">", "430 No Such Article Found")
}

func countBodyRequests(server *nntptest.Server, messageId string) int {
	count := 0
	expectedCmd := "BODY <" + messageId + ">"
	for _, cmd := range server.GetRequestCommands() {
		if cmd == expectedCmd {
			count++
		}
	}
	return count
}

func TestFetchSegment_RoundRobinOnConnectionError(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1): Returns connection error
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	// Server 2 (P2): Returns connection error first, then success
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server2, messageId)
	setupServerWithSegment(t, server2, messageId, testData)
	server2.Start(t)

	// Server 3 (P3): Always returns connection error
	server3 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server3, messageId)
	server3.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})
	pool3 := nntptest.NewPool(t, server3, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: false},
			{Pool: pool3, priority: 0, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify round-robin with retry cycle:
	// Cycle 1: P1 fail -> P2 fail -> P3 fail
	// Cycle 2: P1 fail (excluded) -> P2 success
	assert.Equal(t, 2, countBodyRequests(server1, messageId), "Server1 should receive 2 BODY requests")
	assert.Equal(t, 2, countBodyRequests(server2, messageId), "Server2 should receive 2 BODY requests (1 fail, 1 success)")
	assert.Equal(t, 1, countBodyRequests(server3, messageId), "Server3 should receive 1 BODY request")
}

func TestFetchSegment_PriorityRespected(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, priority 0): Always returns connection error
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	// Server 2 (P2, priority 1): Returns success
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server2, messageId, testData)
	server2.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 1, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify P1 exhausted (2 attempts) before P2 is tried
	assert.Equal(t, 2, countBodyRequests(server1, messageId), "Server1 should receive 2 BODY requests (exhausted)")
	assert.Equal(t, 1, countBodyRequests(server2, messageId), "Server2 should receive 1 BODY request after priority expansion")
}

func TestFetchSegment_BackupNotTriggeredOnConnectionError(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, primary): Always returns connection error
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	// Server 2 (B1, backup): Returns success (but should NOT be called)
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server2, messageId, testData)
	server2.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: true},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	_, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.Error(t, err, "Should return error when all primary providers fail")

	// Verify backup not triggered
	assert.Equal(t, 2, countBodyRequests(server1, messageId), "Server1 should receive 2 BODY requests")
	assert.Equal(t, 0, countBodyRequests(server2, messageId), "Backup server should receive 0 BODY requests")
}

func TestFetchSegment_BackupTriggeredOnArticleNotFound(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, primary): Returns article not found
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerArticleNotFound(server1, messageId)
	server1.Start(t)

	// Server 2 (B1, backup): Returns success
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server2, messageId, testData)
	server2.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: true},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify backup triggered by article-not-found
	assert.Equal(t, 1, countBodyRequests(server1, messageId), "Server1 should receive 1 BODY request")
	assert.Equal(t, 1, countBodyRequests(server2, messageId), "Backup server should receive 1 BODY request")
}

func TestFetchSegment_MixedErrors(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, primary, priority 0): Returns article not found
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerArticleNotFound(server1, messageId)
	server1.Start(t)

	// Server 2 (P2, primary, priority 0): Returns connection error
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server2, messageId)
	server2.Start(t)

	// Server 3 (B1, backup, priority 0): Returns success
	server3 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server3, messageId, testData)
	server3.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})
	pool3 := nntptest.NewPool(t, server3, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: false},
			{Pool: pool3, priority: 0, isBackup: true},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify request distribution
	assert.Equal(t, 1, countBodyRequests(server1, messageId), "Server1 should receive 1 BODY request (article not found)")
	assert.Equal(t, 2, countBodyRequests(server2, messageId), "Server2 should receive 2 BODY requests (connection errors until excluded)")
	assert.Equal(t, 1, countBodyRequests(server3, messageId), "Backup server should receive 1 BODY request")
}

func TestFetchSegment_AllProvidersExhausted(t *testing.T) {
	messageId := "test@example.com"

	// Server 1 (P1): Always returns connection error
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	// Server 2 (P2): Always returns connection error
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server2, messageId)
	server2.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: 100, Number: 1}
	ctx := t.Context()

	_, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.Error(t, err, "Should return error when all providers exhausted")

	// Verify both providers tried maxFailuresPerProvider times
	assert.Equal(t, 2, countBodyRequests(server1, messageId), "Server1 should receive 2 BODY requests")
	assert.Equal(t, 2, countBodyRequests(server2, messageId), "Server2 should receive 2 BODY requests")
}

func TestFetchSegment_MixedPrioritiesExhausted(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, priority 0): Always returns connection error
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	// Server 2 (P2, priority 0): Always returns connection error
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server2, messageId)
	server2.Start(t)

	// Server 3 (P3, priority 1): Always returns connection error
	server3 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server3, messageId)
	server3.Start(t)

	// Server 4 (P4, priority 1): Always returns connection error
	server4 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server4, messageId)
	server4.Start(t)

	// Server 5 (P5, priority 2): Returns success
	server5 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server5, messageId, testData)
	server5.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})
	pool3 := nntptest.NewPool(t, server3, &nntp.PoolConfig{})
	pool4 := nntptest.NewPool(t, server4, &nntp.PoolConfig{})
	pool5 := nntptest.NewPool(t, server5, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: false},
			{Pool: pool3, priority: 1, isBackup: false},
			{Pool: pool4, priority: 1, isBackup: false},
			{Pool: pool5, priority: 2, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify providers exhausted in priority order with round-robin at each level
	// Expected sequence:
	// Priority 0: P1 fail, P2 fail, P1 fail (excluded), P2 fail (excluded)
	// Priority 1: P3 fail, P4 fail, P3 fail (excluded), P4 fail (excluded)
	// Priority 2: P5 success
	assert.Equal(t, 2, countBodyRequests(server1, messageId), "Server1 (priority 0) should receive 2 BODY requests")
	assert.Equal(t, 2, countBodyRequests(server2, messageId), "Server2 (priority 0) should receive 2 BODY requests")
	assert.Equal(t, 2, countBodyRequests(server3, messageId), "Server3 (priority 1) should receive 2 BODY requests")
	assert.Equal(t, 2, countBodyRequests(server4, messageId), "Server4 (priority 1) should receive 2 BODY requests")
	assert.Equal(t, 1, countBodyRequests(server5, messageId), "Server5 (priority 2) should receive 1 BODY request (success)")
}

func TestFetchSegment_BackupMixedPriorities(t *testing.T) {
	testData := []byte("Hello, this is test data")
	messageId := "test@example.com"

	// Server 1 (P1, primary, priority 0): Returns article not found
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerArticleNotFound(server1, messageId)
	server1.Start(t)

	// Server 2 (B1, backup, priority 0): Always returns connection error
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server2, messageId)
	server2.Start(t)

	// Server 3 (B2, backup, priority 1): Returns success
	server3 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerWithSegment(t, server3, messageId, testData)
	server3.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})
	pool3 := nntptest.NewPool(t, server3, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: true},
			{Pool: pool3, priority: 1, isBackup: true},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify backup providers respect priority levels
	// Expected sequence:
	// P1 article not found (triggers backup)
	// Switch to backup priority 0: B1 fail, B1 fail (excluded)
	// Expand to backup priority 1: B2 success
	assert.Equal(t, 1, countBodyRequests(server1, messageId), "Server1 (primary) should receive 1 BODY request (article not found)")
	assert.Equal(t, 2, countBodyRequests(server2, messageId), "Server2 (backup priority 0) should receive 2 BODY requests")
	assert.Equal(t, 1, countBodyRequests(server3, messageId), "Server3 (backup priority 1) should receive 1 BODY request (success)")
}

func TestFetchSegment_ContextCancellation(t *testing.T) {
	messageId := "test@example.com"

	// Server that always returns connection error (to trigger retry loop)
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId)
	server1.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: 100, Number: 1}

	// Create cancelable context and cancel immediately
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.ErrorIs(t, err, context.Canceled, "Should return context.Canceled when context is cancelled")
}

func TestFetchSegment_CacheHit(t *testing.T) {
	testData := []byte("Hello, this is cached data")
	messageId := "cached@example.com"

	// Server should NOT be called
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	setupServerConnectionError(server1, messageId) // Would fail if called
	server1.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})

	// Pre-populate cache
	mockCache := newMockSegmentCache()
	mockCache.Prepopulate(messageId, SegmentData{
		Body:     testData,
		FileSize: int64(len(testData)),
		Size:     int64(len(testData)),
	})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
		},
		segmentCache: mockCache,
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.NoError(t, err)
	assert.Equal(t, testData, data.Body)

	// Verify no server requests made (cache hit)
	assert.Equal(t, 0, countBodyRequests(server1, messageId), "Server should receive 0 BODY requests (cache hit)")
}

func TestFetchSegment_YEncDecodeError(t *testing.T) {
	messageId := "malformed@example.com"

	// Server returns malformed yEnc data
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	malformedBody := []string{"=ybegin line=128 size=100 name=test.bin", "GARBAGE_NOT_YENC_DATA", "=yend size=100"}
	server1.SetResponse("BODY <"+messageId+">", "222 0 <"+messageId+">", malformedBody)
	server1.Start(t)

	// Server 2 should NOT be called (decode error breaks retry loop)
	server2 := nntptest.NewServer(t, "200 NNTP Service Ready")
	testData := []byte("Valid data")
	setupServerWithSegment(t, server2, messageId, testData)
	server2.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})
	pool2 := nntptest.NewPool(t, server2, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
			{Pool: pool2, priority: 0, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: 100, Number: 1}
	ctx := t.Context()

	_, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.Error(t, err, "Should return error on yEnc decode failure")
	assert.NotErrorIs(t, err, ErrArticleNotFound, "Should NOT be article not found error")

	// Verify decode error breaks retry loop (server2 not tried)
	assert.Equal(t, 1, countBodyRequests(server1, messageId), "Server1 should receive 1 BODY request")
	assert.Equal(t, 0, countBodyRequests(server2, messageId), "Server2 should receive 0 BODY requests (decode error breaks loop)")
}

func TestFetchSegment_Singleflight(t *testing.T) {
	testData := []byte("Hello, this is test data for singleflight")
	messageId := "singleflight@example.com"

	// Server with delay to allow concurrent requests to coalesce
	server1 := nntptest.NewServer(t, "200 NNTP Service Ready")
	encoded := encodeYenc(testData, "test.bin", 1, 1, int64(len(testData)), 1)
	lines := strings.Split(strings.TrimSpace(string(encoded)), "\r\n")
	server1.SetResponseWithDelay("BODY <"+messageId+">", 100*time.Millisecond, "222 0 <"+messageId+">", lines)
	server1.Start(t)

	pool1 := nntptest.NewPool(t, server1, &nntp.PoolConfig{})

	usenetPool := &Pool{
		Log: logger.Scoped("test/usenet/pool"),
		providers: []*providerPool{
			{Pool: pool1, priority: 0, isBackup: false},
		},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: int64(len(testData)), Number: 1}
	ctx := t.Context()

	// Launch 5 concurrent goroutines
	var wg sync.WaitGroup
	var successCount atomic.Int32
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
			if err == nil && string(data.Body) == string(testData) {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// All 5 should succeed with same result
	assert.Equal(t, int32(5), successCount.Load(), "All 5 goroutines should receive correct data")

	// Singleflight should coalesce to 1 request
	assert.Equal(t, 1, countBodyRequests(server1, messageId), "Server should receive exactly 1 BODY request (singleflight)")
}

func TestFetchSegment_NoProvidersConfigured(t *testing.T) {
	messageId := "noproviders@example.com"

	usenetPool := &Pool{
		Log:          logger.Scoped("test/usenet/pool"),
		providers:    []*providerPool{},
		segmentCache: getNoopSegmentCache(),
	}

	segment := &nzb.Segment{MessageId: messageId, Bytes: 100, Number: 1}
	ctx := t.Context()

	_, err := usenetPool.fetchSegment(ctx, segment, []string{"alt.test"})
	require.Error(t, err, "Should return error when no providers configured")
	assert.ErrorIs(t, err, ErrNoProvidersConfigured, "Should be ErrNoProvidersConfigured")
}
