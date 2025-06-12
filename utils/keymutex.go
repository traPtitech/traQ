package utils

import "sync"

// KeyMutex キーによるMutex
type KeyMutex struct {
	locks []sync.Mutex
	count uint
}

// NewKeyMutex KeyMutexを生成します
func NewKeyMutex(count uint) *KeyMutex {
	return &KeyMutex{
		count: count,
		locks: make([]sync.Mutex, count),
	}
}

// Lock キーをロックします
func (m *KeyMutex) Lock(key string) {
	m.locks[elfHash(key)%m.count].Lock()
}

// Unlock キーをアンロックします
func (m *KeyMutex) Unlock(key string) {
	m.locks[elfHash(key)%m.count].Unlock()
}

func elfHash(key string) uint {
	h := uint(0)
	for i := range len(key) {
		h = (h << 4) + uint(key[i])
		g := h & 0xF0000000
		if g != 0 {
			h ^= g >> 24
		}
		h &= ^g
	}
	return h
}
