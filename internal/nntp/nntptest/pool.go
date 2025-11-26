package nntptest

import (
	"testing"

	"github.com/MunifTanjim/stremthru/internal/nntp"
)

func NewPool(t *testing.T, server *Server, conf *nntp.PoolConfig) *nntp.Pool {
	t.Helper()

	conf.ConnectionConfig.Host = server.Host()
	conf.ConnectionConfig.Port = server.Port()

	pool, err := nntp.NewPool(conf)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}
