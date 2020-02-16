package sse

import (
	"github.com/gofrs/uuid"
	"sync"
)

type sseClientMap struct {
	sync.Map
}

func (m *sseClientMap) loadClients(key uuid.UUID) (map[uuid.UUID]*sseClient, bool) {
	i, ok := m.Load(key)
	if ok {
		return i.(map[uuid.UUID]*sseClient), true
	}
	return nil, false
}

func (m *sseClientMap) storeClients(key uuid.UUID, value map[uuid.UUID]*sseClient) {
	m.Store(key, value)
}

func (m *sseClientMap) rangeClients(f func(key uuid.UUID, value map[uuid.UUID]*sseClient) bool) {
	m.Range(func(k, v interface{}) bool {
		return f(k.(uuid.UUID), v.(map[uuid.UUID]*sseClient))
	})
}

func (m *sseClientMap) broadcast(data *EventData) {
	m.rangeClients(func(_ uuid.UUID, u map[uuid.UUID]*sseClient) bool {
		for _, c := range u {
			c.RLock()
			skip := c.disconnected
			c.RUnlock()
			if skip {
				continue
			}

			c.send <- data
		}
		return true
	})
}

func (m *sseClientMap) multicast(user uuid.UUID, data *EventData) {
	if u, ok := m.loadClients(user); ok {
		for _, c := range u {
			c.RLock()
			skip := c.disconnected
			c.RUnlock()
			if skip {
				continue
			}

			c.send <- data
		}
	}
}

type sseClient struct {
	sync.RWMutex
	userID       uuid.UUID
	connectionID uuid.UUID
	send         chan *EventData
	disconnected bool
}

func (c *sseClient) dispose() {
	c.Lock()
	c.disconnected = true
	c.Unlock()
	close(c.send)
	// flush buffer
	for range c.send {
	}
}
