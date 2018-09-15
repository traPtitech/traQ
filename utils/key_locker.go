package utils

import (
	"sync"
	"sync/atomic"
)

// KeyLocker KeyによるMutex
type KeyLocker struct {
	inUse sync.Map
	pool  *sync.Pool
}

type refCounter struct {
	counter int64
	lock    *sync.Mutex
}

// Lock キーをロックします
func (l *KeyLocker) Lock(key interface{}) {
	m := l.getLocker(key)
	atomic.AddInt64(&m.counter, 1)
	m.lock.Lock()
}

// Unlock キーをアンロックします
func (l *KeyLocker) Unlock(key interface{}) {
	m := l.getLocker(key)
	m.lock.Unlock()
	atomic.AddInt64(&m.counter, -1)
	if m.counter <= 0 {
		l.pool.Put(m.lock)
		l.inUse.Delete(key)
	}
}

func (l *KeyLocker) getLocker(key interface{}) *refCounter {
	res, _ := l.inUse.LoadOrStore(key, &refCounter{
		counter: 0,
		lock:    l.pool.Get().(*sync.Mutex),
	})
	return res.(*refCounter)
}

// NewKeyLocker KeyLockerを生成します
func NewKeyLocker() *KeyLocker {
	return &KeyLocker{
		pool: &sync.Pool{
			New: func() interface{} {
				return &sync.Mutex{}
			},
		},
	}
}
