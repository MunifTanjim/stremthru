package usenetmanager

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	usenet_server "github.com/MunifTanjim/stremthru/internal/usenet/server"
)

var ErrServerLocked = errors.New("server is locked for modification")

var globalManager = &Manager{
	lockedServers: make(map[string]struct{}),
}

var getSegmentCache = sync.OnceValue(func() usenet_pool.SegmentCache {
	return usenet_pool.NewSegmentCache(config.NewzSegmentCacheSize)
})

type Manager struct {
	pool      *usenet_pool.Pool
	poolMutex sync.RWMutex

	log      *logger.Logger
	initOnce sync.Once

	closed atomic.Bool

	lockedServers      map[string]struct{}
	lockedServersMutex sync.RWMutex

	pendingTimers      []*time.Timer
	pendingTimersMutex sync.Mutex
}

func (m *Manager) getPool() *usenet_pool.Pool {
	m.poolMutex.RLock()
	defer m.poolMutex.RUnlock()
	return m.pool
}

func (m *Manager) getProviderAcquiredCount(serverId string) int {
	pool := m.getPool()
	if pool == nil {
		return 0
	}
	return pool.GetAcquiredConnectionCount(serverId)
}

func GetPool(log *logger.Logger) (*usenet_pool.Pool, error) {
	if log == nil {
		log = logger.Scoped("usenet/pool/manager")
	}

	var initErr error
	globalManager.initOnce.Do(func() {
		globalManager.log = log
		initErr = globalManager.initialize()
	})

	if initErr != nil {
		return nil, initErr
	}

	return globalManager.getPool(), nil
}

func LockServer(serverId string) error {
	globalManager.lockedServersMutex.Lock()
	defer globalManager.lockedServersMutex.Unlock()

	if _, locked := globalManager.lockedServers[serverId]; locked {
		return ErrServerLocked
	}

	if globalManager.getProviderAcquiredCount(serverId) > 0 {
		return ErrServerLocked
	}

	globalManager.lockedServers[serverId] = struct{}{}
	if globalManager.log != nil {
		globalManager.log.Debug("server locked", "server_id", serverId)
	}
	return nil
}

func UnlockServer(serverId string) {
	globalManager.lockedServersMutex.Lock()
	defer globalManager.lockedServersMutex.Unlock()

	delete(globalManager.lockedServers, serverId)
	if globalManager.log != nil {
		globalManager.log.Debug("server unlocked", "server_id", serverId)
	}
}

func IsServerLocked(serverId string) bool {
	globalManager.lockedServersMutex.RLock()
	defer globalManager.lockedServersMutex.RUnlock()
	_, locked := globalManager.lockedServers[serverId]
	return locked
}

func IsServerInUse(serverId string) bool {
	if IsServerLocked(serverId) {
		return true
	}
	return globalManager.getProviderAcquiredCount(serverId) > 0
}

func Close() {
	if globalManager.closed.Swap(true) {
		return
	}

	globalManager.cancelPendingTimers()
	globalManager.closePool()

	if globalManager.log != nil {
		globalManager.log.Info("global NNTP pool manager closed")
	}
}

func (m *Manager) cancelPendingTimers() {
	m.pendingTimersMutex.Lock()
	defer m.pendingTimersMutex.Unlock()

	for _, timer := range m.pendingTimers {
		timer.Stop()
	}
	m.pendingTimers = nil
}

func (m *Manager) closePool() {
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	if m.pool != nil {
		m.pool.Close()
		m.pool = nil
	}
}

func (m *Manager) initialize() error {
	m.log.Info("initializing global NNTP pool")
	return m.rebuildPool()
}

func (m *Manager) rebuildPool() error {
	servers, err := usenet_server.GetAll()
	if err != nil {
		m.log.Error("failed to get servers from vault", "error", err)
		return err
	}

	newPool, err := m.createPoolFromServers(servers)
	if err != nil {
		m.log.Error("failed to create pool", "error", err)
		return err
	}

	oldPool := m.swapPool(newPool)
	m.clearServerLocks()

	m.log.Info("pool rebuilt", "server_count", len(servers))

	if oldPool != nil {
		m.schedulePoolCleanup(oldPool)
	}

	return nil
}

func (m *Manager) swapPool(newPool *usenet_pool.Pool) *usenet_pool.Pool {
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	oldPool := m.pool
	m.pool = newPool
	return oldPool
}

func (m *Manager) clearServerLocks() {
	m.lockedServersMutex.Lock()
	defer m.lockedServersMutex.Unlock()
	m.lockedServers = make(map[string]struct{})
}

func (m *Manager) schedulePoolCleanup(pool *usenet_pool.Pool) {
	timer := time.AfterFunc(30*time.Second, func() {
		if m.closed.Load() {
			return
		}
		m.log.Debug("closing old pool")
		pool.Close()
	})

	m.pendingTimersMutex.Lock()
	m.pendingTimers = append(m.pendingTimers, timer)
	m.pendingTimersMutex.Unlock()
}

func (m *Manager) createEmptyPool() (*usenet_pool.Pool, error) {
	return usenet_pool.NewPool(&usenet_pool.Config{
		Log:          m.log,
		Providers:    []usenet_pool.ProviderConfig{},
		SegmentCache: getSegmentCache(),
	})
}

func (m *Manager) createPoolFromServers(servers []usenet_server.UsenetServer) (*usenet_pool.Pool, error) {
	if len(servers) == 0 {
		m.log.Warn("no servers configured, creating empty pool")
		return m.createEmptyPool()
	}

	providers := make([]usenet_pool.ProviderConfig, 0, len(servers))
	for i := range servers {
		s := &servers[i]
		password, err := s.GetPassword()
		if err != nil {
			m.log.Warn("failed to decrypt password", "server", s.Name, "error", err)
			continue
		}

		providers = append(providers, usenet_pool.ProviderConfig{
			PoolConfig: nntp.PoolConfig{
				ConnectionConfig: nntp.ConnectionConfig{
					Host:          s.Host,
					Port:          s.Port,
					Username:      s.Username,
					Password:      password,
					TLS:           s.TLS,
					TLSSkipVerify: s.TLSSkipVerify,
				},
				MaxSize: int32(s.MaxConnections),
			},
			Priority: s.Priority,
			IsBackup: s.IsBackup,
		})
	}

	if len(providers) == 0 {
		m.log.Warn("no valid providers after password decryption")
		return m.createEmptyPool()
	}

	return usenet_pool.NewPool(&usenet_pool.Config{
		Log:          m.log,
		Providers:    providers,
		SegmentCache: getSegmentCache(),
	})
}

func (m *Manager) createProviderConfig(server *usenet_server.UsenetServer) (*usenet_pool.ProviderConfig, error) {
	password, err := server.GetPassword()
	if err != nil {
		return nil, err
	}

	return &usenet_pool.ProviderConfig{
		PoolConfig: nntp.PoolConfig{
			ConnectionConfig: nntp.ConnectionConfig{
				Host:          server.Host,
				Port:          server.Port,
				Username:      server.Username,
				Password:      password,
				TLS:           server.TLS,
				TLSSkipVerify: server.TLSSkipVerify,
			},
			MaxSize: int32(server.MaxConnections),
		},
		Priority: server.Priority,
		IsBackup: server.IsBackup,
	}, nil
}

func AddServer(providerId string) error {
	server, err := usenet_server.GetById(providerId)
	if err != nil {
		return err
	}
	if server == nil {
		return nil
	}

	pool := globalManager.getPool()
	if pool == nil {
		return nil
	}

	config, err := globalManager.createProviderConfig(server)
	if err != nil {
		if globalManager.log != nil {
			globalManager.log.Warn("failed to create provider config", "server", server.Name, "error", err)
		}
		return err
	}

	return pool.AddProvider(config)
}

func RemoveServer(serverId string) {
	pool := globalManager.getPool()
	if pool == nil {
		return
	}

	pool.RemoveProvider(serverId)
	UnlockServer(serverId)
}

func UpdateServer(oldServerId, newServerId string) error {
	pool := globalManager.getPool()
	if pool == nil {
		return nil
	}

	pool.RemoveProvider(oldServerId)
	UnlockServer(oldServerId)

	server, err := usenet_server.GetById(newServerId)
	if err != nil {
		return err
	}
	if server == nil {
		return nil
	}

	config, err := globalManager.createProviderConfig(server)
	if err != nil {
		if globalManager.log != nil {
			globalManager.log.Warn("failed to create provider config", "error", err, "server", server.Name)
		}
		return err
	}

	return pool.AddProvider(config)
}
