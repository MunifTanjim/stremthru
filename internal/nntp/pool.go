package nntp

import (
	"context"
	"errors"
	"math"
	"net/textproto"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/jackc/puddle/v2"
)

type PoolConfig struct {
	ConnectionConfig

	Log *logger.Logger

	MinSize int32
	MaxSize int32

	HealthCheckTimeout time.Duration
	ReconnectTimeout   time.Duration
	ReconnectDelay     time.Duration
}

func (c *PoolConfig) Id() string {
	return c.Host + ":" + strconv.Itoa(c.Port) + ":" + c.Username
}

func (c *PoolConfig) setDefaults() {
	if c.Log == nil {
		c.Log = logger.Scoped("nntp/pool")
	}
	if c.MinSize < 0 {
		c.MinSize = 0
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 10
	}
	if c.MinSize > c.MaxSize {
		c.MinSize = c.MaxSize
	}
	if c.HealthCheckTimeout <= 0 {
		c.HealthCheckTimeout = 10 * time.Second
	}
	if c.ReconnectTimeout <= 0 {
		c.ReconnectTimeout = 30 * time.Second
	}
	if c.ReconnectDelay <= 0 {
		c.ReconnectDelay = 1 * time.Minute
	}
}

type PoolState string

const (
	PoolStateAuthFailed PoolState = "auth_failed"
	PoolStateConnecting PoolState = "connecting"
	PoolStateOffline    PoolState = "offline"
	PoolStateOnline     PoolState = "online"
	PoolStateDisabled   PoolState = "disabled"
)

type Pool struct {
	Log *logger.Logger

	id     string
	pool   *puddle.Pool[*Connection]
	config *PoolConfig

	closeCh chan struct{}
	closed  atomic.Bool

	state      PoolState
	stateMutex sync.RWMutex
	wg         sync.WaitGroup

	reconnectScheduled atomic.Bool
}

func (p *Pool) Id() string {
	if p.id == "" {
		p.id = p.config.Id()
	}
	return p.id
}

func (p *Pool) GetState() PoolState {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return p.state
}

func (p *Pool) SetState(state PoolState) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	p.state = state
}

func (p *Pool) MaxSize() int32 {
	return p.config.MaxSize
}

func (p *Pool) MinSize() int32 {
	return p.config.MinSize
}

func NewPool(config *PoolConfig) (*Pool, error) {
	if config.Host == "" {
		panic("nntp: missing host")
	}

	config.setDefaults()

	p := &Pool{
		Log:     config.Log,
		config:  config,
		state:   PoolStateOnline,
		closeCh: make(chan struct{}),
	}

	constructor := func(ctx context.Context) (*Connection, error) {
		conn := &Connection{}
		if err := conn.Connect(ctx, &config.ConnectionConfig); err != nil {
			return nil, err
		}
		return conn, nil
	}

	destructor := func(conn *Connection) {
		conn.Close()
	}

	poolConfig := &puddle.Config[*Connection]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     config.MaxSize,
	}

	pool, err := puddle.NewPool(poolConfig)
	if err != nil {
		return nil, err
	}

	p.pool = pool

	if config.MinSize > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := p.ensureMinSize(ctx); err != nil {
			p.Log.Warn("failed to ensure min size at startup", "error", err)
		}
	}

	return p, nil
}

func (p *Pool) ensureMinSize(ctx context.Context) error {
	totalCount := p.pool.Stat().TotalResources()
	for range p.config.MinSize - totalCount {
		c, err := p.Acquire(ctx)
		if err != nil {
			return err
		}
		c.Release()
	}
	return nil
}

func (p *Pool) handleConnectionFailure(errs ...error) {
	p.Log.Trace("handleConnectionFailure", "error_count", len(errs))

	for _, err := range errs {
		var nntpErr *Error
		if errors.As(err, &nntpErr) {
			if nntpErr.isAuthError() {
				p.Log.Trace("handleConnectionFailure auth error", "error", nntpErr)
				p.SetState(PoolStateAuthFailed)
				return
			}
		}
	}

	currentState := p.GetState()

	if currentState == PoolStateOnline || currentState == PoolStateConnecting {
		p.Log.Trace("handleConnectionFailure setting offline", "previous_state", currentState)
		p.SetState(PoolStateOffline)
		if currentState == PoolStateOnline {
			p.destroyAllIdles()
		}
		p.scheduleReconnect()
	}
}

func (p *Pool) IsOnline() bool {
	state := p.GetState()
	return state == PoolStateOnline
}

func (p *Pool) doReconnect(ctx context.Context) bool {
	switch p.GetState() {
	case PoolStateDisabled, PoolStateAuthFailed:
		return false
	}

	p.SetState(PoolStateConnecting)

	reconnectCtx, cancelReconnectCtx := context.WithTimeout(ctx, p.config.ReconnectTimeout)
	defer cancelReconnectCtx()

	c, err := p.pool.Acquire(reconnectCtx)
	if err != nil {
		p.Log.Warn("reconnection attempt failed", "host", p.config.Host, "error", err)
		p.SetState(PoolStateOffline)
		return false
	}

	defer c.Release()

	p.SetState(PoolStateOnline)
	return true
}

func (p *Pool) scheduleReconnect() {
	if !p.reconnectScheduled.CompareAndSwap(false, true) {
		return
	}

	maxInterval := time.Duration(math.Pow(2, 4)) * p.config.ReconnectDelay

	p.wg.Go(func() {
		defer p.reconnectScheduled.Store(false)

		p.Log.Debug("starting reconnection loop", "id", p.Id())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		interval := p.config.ReconnectDelay
		attempt := 0

		for {
			select {
			case <-p.closeCh:
				p.Log.Debug("reconnection loop stopped", "id", p.Id(), "reason", "stop signal")
				return
			case <-time.After(interval):
				if p.closed.Load() {
					p.Log.Debug("reconnection loop stopped", "id", p.Id(), "reason", "pool closed")
					return
				}

				state := p.GetState()
				if state != PoolStateOffline {
					p.Log.Debug("reconnection loop stopped", "id", p.Id(), "reason", "state changed", "state", state)
					return
				}

				attempt++
				p.Log.Debug("attempting reconnection", "id", p.Id(), "attempt", attempt, "backoff", interval)

				if p.doReconnect(ctx) {
					p.Log.Debug("reconnection successful", "id", p.Id())
					return
				}

				interval = min(interval*2, maxInterval)
				p.Log.Debug("reconnection failed, backing off", "id", p.Id(), "backoff", interval)
			}
		}
	})
}

func (p *Pool) destroyAllIdles() {
	for _, conn := range p.pool.AcquireAllIdle() {
		conn.Destroy()
	}
}

func (p *Pool) Acquire(ctx context.Context) (*PooledConnection, error) {
	const maxRetries = 3

	errs := []error{}
	for attempt := range maxRetries {
		currState := p.GetState()
		if currState != PoolStateOnline && currState != PoolStateConnecting {
			return nil, ErrPoolNotOnline
		}

		p.Log.Trace("Acquire - connection acquiring", "state", currState, "attempt", attempt+1)

		var conn *PooledConnection

		res, err := p.pool.Acquire(ctx)
		if err == nil {
			conn = &PooledConnection{
				Connection: res.Value(),
				resource:   res,
				pool:       p,
			}

			// Health check using DATE command - simple, read-only, and widely supported.
			// Note: DATE is optional per RFC 3977 but supported by most usenet servers.
			p.Log.Trace("Acquire - health check", "provider", p.Id())
			_, err = conn.Date()
		}

		if err == nil {
			p.Log.Trace("Acquire - connection acquired", "provider", p.Id())
			return conn, nil
		} else {
			var tpErr *textproto.Error
			var nntpErr *Error
			if !errors.As(err, &tpErr) && !errors.As(err, &nntpErr) {
				if isConnectionError(err) {
					p.Log.Trace("Acquire - connection error, retrying", "error", err, "attempt", attempt+1)
					conn.Destroy()
					errs = append(errs, err)
					continue
				}
			}
			p.handleConnectionFailure(err)
			return nil, err
		}
	}

	p.handleConnectionFailure(errs...)
	return nil, NewConnectionError("failed to acquire healthy connection after max retries")
}

func (p *Pool) AcquireForGroup(ctx context.Context, group string) (*PooledConnection, error) {
	pc, err := p.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	if pc.currentGroup == group {
		return pc, nil
	}

	_, err = pc.Group(group)
	if err != nil {
		pc.Destroy()
		return nil, err
	}

	return pc, nil
}

func (p *Pool) Stat() *puddle.Stat {
	return p.pool.Stat()
}

func (p *Pool) Close() {
	if p.closed.Swap(true) {
		return
	}

	close(p.closeCh)

	p.wg.Wait()
	p.pool.Close()
}

type PooledConnection struct {
	*Connection
	resource *puddle.Resource[*Connection]
	pool     *Pool
	released atomic.Bool
}

func (pc *PooledConnection) Release() {
	if pc.released.Swap(true) {
		return
	}
	pc.resource.Release()
}

func (pc *PooledConnection) Destroy() {
	if pc.released.Swap(true) {
		return
	}
	pc.resource.Destroy()
}

func (pc *PooledConnection) Hijack() *Connection {
	if pc.released.Swap(true) {
		return nil
	}
	pc.resource.Hijack()
	return pc.Connection
}

func (pc *PooledConnection) CurrentGroup() string {
	return pc.currentGroup
}

func (pc *PooledConnection) ProviderId() string {
	return pc.pool.Id()
}
