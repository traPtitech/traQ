package viewer

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"sync"
	"time"
)

// Manager チャンネル閲覧者マネージャ
type Manager struct {
	hub      *hub.Hub
	channels map[uuid.UUID]map[*viewer]struct{}
	viewers  map[interface{}]*viewer
	mu       sync.RWMutex
}

type viewer struct {
	key       interface{}
	userID    uuid.UUID
	channelID uuid.UUID
	state     State
}

// NewManager チャンネル閲覧者マネージャーを生成します
func NewManager(hub *hub.Hub) *Manager {
	vm := &Manager{
		hub:      hub,
		channels: map[uuid.UUID]map[*viewer]struct{}{},
		viewers:  map[interface{}]*viewer{},
	}

	go func() {
		for range time.NewTicker(5 * time.Minute).C {
			vm.mu.Lock()
			vm.gc()
			vm.mu.Unlock()
		}
	}()
	return vm
}

// GetChannelViewers 指定したチャンネルのチャンネル閲覧者状態を取得します
func (vm *Manager) GetChannelViewers(channelID uuid.UUID) map[uuid.UUID]State {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return calculateChannelViewers(vm.channels[channelID])
}

// SetViewer 指定したキーのチャンネル閲覧者状態を設定します
func (vm *Manager) SetViewer(key interface{}, userID uuid.UUID, channelID uuid.UUID, state State) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	cv, ok := vm.channels[channelID]
	if !ok {
		cv = map[*viewer]struct{}{}
		vm.channels[channelID] = cv
	}

	v, ok := vm.viewers[key]
	if ok {
		if v.channelID == channelID {
			if v.state == state {
				// 何も変わってない
				return
			}
			// stateだけ変更
			v.state = state
		} else {
			// channelとstateが変更
			oldC := v.channelID
			old := vm.channels[oldC]
			delete(old, v)

			v.channelID = channelID
			v.state = state

			vm.hub.Publish(hub.Message{
				Name: event.ChannelViewersChanged,
				Fields: hub.Fields{
					"channel_id": oldC,
					"viewers":    calculateChannelViewers(old),
				},
			})
		}
	} else {
		v = &viewer{
			key:       key,
			userID:    userID,
			channelID: channelID,
			state:     state,
		}
		vm.viewers[key] = v
	}

	cv[v] = struct{}{}
	vm.hub.Publish(hub.Message{
		Name: event.ChannelViewersChanged,
		Fields: hub.Fields{
			"channel_id": channelID,
			"viewers":    calculateChannelViewers(cv),
		},
	})
}

// RemoveViewer 指定したキーのチャンネル閲覧者状態を削除します
func (vm *Manager) RemoveViewer(key interface{}) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	v, ok := vm.viewers[key]
	if !ok {
		return
	}

	cv := vm.channels[v.channelID]
	delete(vm.viewers, key)
	delete(cv, v)

	vm.hub.Publish(hub.Message{
		Name: event.ChannelViewersChanged,
		Fields: hub.Fields{
			"channel_id": v.channelID,
			"viewers":    calculateChannelViewers(cv),
		},
	})
}

// 5分に１回呼び出される。チャンネルマップのお掃除
func (vm *Manager) gc() {
	for cid, cv := range vm.channels {
		if len(cv) == 0 {
			delete(vm.channels, cid)
		}
	}
}

func calculateChannelViewers(vs map[*viewer]struct{}) map[uuid.UUID]State {
	result := make(map[uuid.UUID]State, len(vs))
	for v := range vs {
		if s, ok := result[v.userID]; ok && s > v.state {
			continue
		}
		result[v.userID] = v.state
	}
	return result
}
