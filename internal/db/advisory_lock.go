package db

import (
	"errors"
	"hash/crc32"
	"strings"
	"sync"

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
	name  string
	count int
	m     sync.Mutex
}

func (l *sqliteAdvisoryLock) GetName() string {
	return l.name
}

func (l *sqliteAdvisoryLock) Acquire() bool {
	l.count++
	// lockLog.Debug("acquired", "name", l.name, "count", l.count)
	return true
}

func (l *sqliteAdvisoryLock) TryAcquire() bool {
	l.count++
	// lockLog.Debug("acquired", "name", l.name, "count", l.count)
	return true
}

func (l *sqliteAdvisoryLock) Release() bool {
	if l.count == 0 {
		return false
	}
	l.count--
	// lockLog.Debug("released", "name", l.name, "count", l.count)
	return true
}

func (l *sqliteAdvisoryLock) ReleaseAll() bool {
	if l.count == 0 {
		return false
	}
	l.count = 0
	// lockLog.Debug("released all", "name", l.name, "count", l.count)
	return true
}

func (l *sqliteAdvisoryLock) Err() error {
	return nil
}

func sqliteNewAdvisoryLock(names ...string) AdvisoryLock {
	return &sqliteAdvisoryLock{
		Executor: db,
		name:     strings.Join(names, ":"),
	}
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

func (l *postgresAdvisoryLock) GetName() string {
	return l.name
}

func (l *postgresAdvisoryLock) Acquire() bool {
	_, err := l.Exec("SELECT pg_advisory_lock(?, ?)", l.keyA, l.keyB)
	if err != nil {
		lockLog.Error("acquire failed", "error", err, "name", l.name)
		l.err = errors.Join(l.err, err)
		return false
	}
	l.count++
	// lockLog.Debug("acquired", "name", l.name, "count", l.count)
	return true
}

func (l *postgresAdvisoryLock) TryAcquire() bool {
	row := l.QueryRow("SELECT pg_try_advisory_lock(?, ?)", l.keyA, l.keyB)
	var acquired bool
	if err := row.Scan(&acquired); err != nil {
		l.err = errors.Join(l.err, err)
		lockLog.Error("try acquire failed", "error", l.err, "name", l.name)
		return false
	} else if !acquired {
		lockLog.Debug("try acquire failed", "name", l.name, "count", l.count)
		return false
	}
	l.count++
	// lockLog.Debug("try acquired", "name", l.name, "count", l.count)
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
	// lockLog.Debug("released", "name", l.name, "count", l.count)
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
	// lockLog.Debug("released all", "name", l.name, "count", l.count)
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
