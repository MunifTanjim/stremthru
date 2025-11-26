package nntp_test

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool_DefaultValues(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	assert.Equal(t, int32(10), pool.MaxSize())
}

func TestNewPool_CustomValues(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 5,
	})
	require.NoError(t, err)
	defer pool.Close()

	assert.Equal(t, int32(5), pool.MaxSize())
}

func TestNewPool_PanicOnMissingHost(t *testing.T) {
	assert.Panics(t, func() {
		NewPool(&PoolConfig{})
	})
}

func TestPool_Acquire(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.Connection)

	stats := pool.Stat()
	assert.Equal(t, int32(1), stats.AcquiredResources())

	conn.Release()

	stats = pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredResources())
	assert.Equal(t, int32(1), stats.IdleResources())
}

func TestPool_AcquireMultiple(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 3,
	})
	require.NoError(t, err)
	defer pool.Close()

	conns := make([]*PooledConnection, 3)
	for i := range 3 {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		conns[i] = conn
	}

	stats := pool.Stat()
	assert.Equal(t, int32(3), stats.AcquiredResources())

	for _, conn := range conns {
		conn.Release()
	}

	stats = pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredResources())
	assert.Equal(t, int32(3), stats.IdleResources())
}

func TestPool_AcquireForGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("GROUP *", "211 100 1 100 alt.test")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.AcquireForGroup(ctx, "alt.test")
	require.NoError(t, err)
	assert.Equal(t, "alt.test", conn.CurrentGroup())

	conn.Release()

	// Acquire again - should reuse connection with same group
	conn2, err := pool.AcquireForGroup(ctx, "alt.test")
	require.NoError(t, err)
	assert.Equal(t, "alt.test", conn2.CurrentGroup())

	conn2.Release()
}

func TestPool_AcquireForGroup_SwitchesGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("GROUP alt.test", "211 100 1 100 alt.test")
	server.SetResponse("GROUP alt.other", "211 50 1 50 alt.other")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 1, // Force reuse
	})
	require.NoError(t, err)
	defer pool.Close()

	conn1, err := pool.AcquireForGroup(ctx, "alt.test")
	require.NoError(t, err)
	assert.Equal(t, "alt.test", conn1.CurrentGroup())
	conn1.Release()

	// Acquire for different group - should switch
	conn2, err := pool.AcquireForGroup(ctx, "alt.other")
	require.NoError(t, err)
	assert.Equal(t, "alt.other", conn2.CurrentGroup())
	conn2.Release()
}

func TestPool_AcquireForGroup_NoSuchGroup(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("GROUP *", "411 No such group")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.AcquireForGroup(ctx, "nonexistent.group")
	require.Error(t, err)

	nntpErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, ErrorCodeNoSuchGroup, nntpErr.Code)

	// Wait for async destruction
	time.Sleep(50 * time.Millisecond)

	// Connection should be destroyed on error (not returned to pool)
	stats := pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredResources())
	assert.Equal(t, int32(0), stats.TotalResources())
}

func TestPooledConnection_Release_Idempotent(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	// Multiple releases should be safe
	conn.Release()
	conn.Release()
	conn.Release()

	stats := pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredResources())
}

func TestPooledConnection_Destroy(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	stats := pool.Stat()
	assert.Equal(t, int32(1), stats.TotalResources())

	conn.Destroy()

	// Wait briefly for async destruction
	time.Sleep(50 * time.Millisecond)

	stats = pool.Stat()
	assert.Equal(t, int32(0), stats.TotalResources())
}

func TestPooledConnection_Hijack(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)

	hijacked := conn.Hijack()
	assert.NotNil(t, hijacked)

	// Second hijack should return nil
	hijacked2 := conn.Hijack()
	assert.Nil(t, hijacked2)

	stats := pool.Stat()
	assert.Equal(t, int32(0), stats.TotalResources())

	// Must close hijacked connection manually
	hijacked.Close()
}

func TestPool_ConcurrentAcquire(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 5,
	})
	require.NoError(t, err)
	defer pool.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	for range 10 {
		wg.Go(func() {
			conn, err := pool.Acquire(ctx)
			if err != nil {
				errCh <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
			conn.Release()
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent acquire error: %v", err)
	}
}

func TestPool_AcquireWithCanceledContext(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	poolCtx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 1,
	})
	require.NoError(t, err)
	defer pool.Close()

	// Acquire the only available connection
	conn, err := pool.Acquire(poolCtx)
	require.NoError(t, err)

	// Try to acquire with a canceled context
	acquireCtx, acquireCancel := context.WithCancel(context.Background())
	acquireCancel() // Cancel immediately

	_, err = pool.Acquire(acquireCtx)
	assert.Error(t, err)

	conn.Release()
}

func TestPool_Close(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
	})
	require.NoError(t, err)

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	conn.Release()

	stats := pool.Stat()
	assert.Equal(t, int32(1), stats.TotalResources())

	pool.Close()

	stats = pool.Stat()
	assert.Equal(t, int32(0), stats.TotalResources())
}

func TestPool_Stats(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host: server.Host(),
			Port: server.Port(),
		},
		MaxSize: 5,
	})
	require.NoError(t, err)
	defer pool.Close()

	stats := pool.Stat()
	assert.Equal(t, int32(5), stats.MaxResources())
	assert.Equal(t, int32(0), stats.AcquiredResources())

	conn1, _ := pool.Acquire(ctx)
	conn2, _ := pool.Acquire(ctx)

	stats = pool.Stat()
	assert.Equal(t, int32(2), stats.AcquiredResources())
	assert.GreaterOrEqual(t, stats.AcquireCount(), int64(2))

	conn1.Release()
	conn2.Release()

	stats = pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredResources())
	assert.Equal(t, int32(2), stats.IdleResources())
}

func TestPool_WithAuthentication(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("AUTHINFO USER testuser", "381 Password required")
	server.SetResponse("AUTHINFO PASS testpass", "281 Authentication accepted")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host:     server.Host(),
			Port:     server.Port(),
			Username: "testuser",
			Password: "testpass",
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	reqCmds := server.GetRequestCommands()
	assert.True(t, reqCmds.HasCommand("AUTHINFO USER testuser"))
	assert.True(t, reqCmds.HasCommand("AUTHINFO PASS testpass"))
	conn.Release()
}

func TestPool_AuthenticationFailed(t *testing.T) {
	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("AUTHINFO USER testuser", "381 Password required")
	server.SetResponse("AUTHINFO PASS wrongpass", "481 Authentication failed")
	server.Start(t)

	ctx := t.Context()

	pool, err := NewPool(&PoolConfig{
		ConnectionConfig: ConnectionConfig{
			Host:     server.Host(),
			Port:     server.Port(),
			Username: "testuser",
			Password: "wrongpass",
		},
	})
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Acquire(ctx)
	assert.Error(t, err)

	nntpErr, ok := err.(*Error)
	require.True(t, ok)
	assert.Equal(t, ErrorCodeAuthentication, nntpErr.Code)
}
