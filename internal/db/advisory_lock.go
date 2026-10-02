package db

import (
	"errors"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
)

var lockLog = logger.Scoped("db/advisory_lock")

var stremthruChecksum = crc32.ChecksumIEEE([]byte("STREMTHRU"))

func getAdvisoryLockKeyPair(names ...string) (uint32, uint32) {
	return stremthruChecksum, crc32.ChecksumIEEE([]byte(strings.Join(names, string(rune(0)))))
}

type AdvisoryLock interface {
	Executor
	GetName() string
	Acquire() bool
	TryAcquire() bool
	Release() bool
	ReleaseAll() bool
	Err() error
}

type sqliteAdvisoryLock struct {
	Executor
	name   string
	locked bool
	m      sync.Mutex
}

var sqliteAdvisoryLockByName sync.Map

func (l *sqliteAdvisoryLock) lock() bool {
	l.m.Lock()
	defer l.m.Unlock()
	if !l.locked {
		l.locked = true
	}
	return l.locked
}

func (l *sqliteAdvisoryLock) unlock() bool {
	l.m.Lock()
	defer l.m.Unlock()
	if l.locked {
		l.locked = false
	}
	return !l.locked
}

func (l *sqliteAdvisoryLock) GetName() string {
	return l.name
}

func (l *sqliteAdvisoryLock) Acquire() bool {
	tryLeft := 5
	for tryLeft > 0 && !l.lock() {
		tryLeft--
		time.Sleep(1 * time.Second)
	}
	return l.locked
}

func (l *sqliteAdvisoryLock) TryAcquire() bool {
	return l.lock()
}

func (l *sqliteAdvisoryLock) Release() bool {
	return l.unlock()
}

func (l *sqliteAdvisoryLock) ReleaseAll() bool {
	return l.Release()
}

func (l *sqliteAdvisoryLock) Err() error {
	return nil
}

func sqliteNewAdvisoryLock(names ...string) AdvisoryLock {
	name := strings.Join(names, ":")
	if lock, ok := sqliteAdvisoryLockByName.Load(name); ok {
		return lock.(*sqliteAdvisoryLock)
	}
	lock := &sqliteAdvisoryLock{Executor: db, name: name}
	sqliteAdvisoryLockByName.Store(name, lock)
	return lock
}

type postgresAdvisoryLock struct {
	Executor
	name  string
	count int
	err   error
	keyA  int32
	keyB  int32
}

func (l *postgresAdvisoryLock) commit() {
	if l.Executor == nil {
		return
	}
	err := l.Executor.(*Tx).Commit()
	if err != nil {
		lockLog.Error("lock tx commit failed", "error", err, "name", l.name)
		return
	}
	l.Executor = nil
}

// releases the tx (and its pooled connection) when no lock is held
func (l *postgresAdvisoryLock) rollbackIfUnlocked() {
	if l.count != 0 || l.Executor == nil {
		return
	}
	err := l.Executor.(*Tx).Rollback()
	if err != nil {
		lockLog.Error("lock tx rollback failed", "error", err, "name", l.name)
	}
	l.Executor = nil
}

func (l *postgresAdvisoryLock) GetName() string {
	return l.name
}

func (l *postgresAdvisoryLock) Acquire() bool {
	if l.Executor == nil {
		lockLog.Error("acquire failed, lock tx closed", "name", l.name)
		return false
	}
	_, err := l.Exec("SELECT pg_advisory_lock(?, ?)", l.keyA, l.keyB)
	if err != nil {
		lockLog.Error("acquire failed", "error", err, "name", l.name)
		l.err = errors.Join(l.err, err)
		l.rollbackIfUnlocked()
		return false
	}
	l.count++
	return true
}

func (l *postgresAdvisoryLock) TryAcquire() bool {
	if l.Executor == nil {
		lockLog.Error("try acquire failed, lock tx closed", "name", l.name)
		return false
	}
	row := l.QueryRow("SELECT pg_try_advisory_lock(?, ?)", l.keyA, l.keyB)
	var acquired bool
	if err := row.Scan(&acquired); err != nil {
		l.err = errors.Join(l.err, err)
		lockLog.Error("try acquire failed", "error", l.err, "name", l.name)
		l.rollbackIfUnlocked()
		return false
	} else if !acquired {
		lockLog.Debug("try acquire failed", "name", l.name, "count", l.count)
		l.rollbackIfUnlocked()
		return false
	}
	l.count++
	return acquired
}

func (l *postgresAdvisoryLock) Release() bool {
	if l.count == 0 {
		l.commit()
		return false
	}
	row := l.QueryRow("SELECT pg_advisory_unlock(?, ?)", l.keyA, l.keyB)
	var released bool
	if err := row.Scan(&released); err != nil {
		l.err = errors.Join(l.err, err)
		lockLog.Error("release failed", "error", l.err, "name", l.name)
		return false
	} else if !released {
		lockLog.Debug("release failed", "name", l.name, "count", l.count)
		return false
	}
	l.count--
	if l.count == 0 {
		l.commit()
	}
	return true
}

func (l *postgresAdvisoryLock) ReleaseAll() bool {
	if l.count == 0 {
		l.commit()
		return false
	}
	for range l.count {
		if !l.Release() {
			break
		}
	}
	if l.count != 0 {
		lockLog.Error("release all failed", "name", l.name, "count", l.count)
		return false
	}
	return true
}

func (l *postgresAdvisoryLock) Err() error {
	return l.err
}

func postgresNewAdvisoryLock(names ...string) AdvisoryLock {
	name := strings.Join(names, ":")
	tx, err := Begin()
	if err != nil {
		lockLog.Error("lock tx begin failed", "error", err, "name", name)
		return nil
	}
	keyA, keyB := getAdvisoryLockKeyPair(names...)
	return &postgresAdvisoryLock{
		Executor: tx,
		name:     name,
		keyA:     int32(keyA),
		keyB:     int32(keyB),
	}
}
