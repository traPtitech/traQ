package utils

import "sync"

// KeyLocker KeyによるMutex
type KeyLocker struct {
	inUse sync.Map
	pool  *sync.Pool
}

// Lock キーをロックします
func (l *KeyLocker) Lock(key interface{}) {
	l.getLocker(key).Lock()
}

// Unlock キーをアンロックします
func (l *KeyLocker) Unlock(key interface{}) {
	m := l.getLocker(key)
	m.Unlock()
	l.pool.Put(m)
	l.inUse.Delete(key)
}

func (l *KeyLocker) getLocker(key interface{}) *sync.Mutex {
	res, _ := l.inUse.LoadOrStore(key, l.pool.Get().(*sync.Mutex))
	return res.(*sync.Mutex)
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
