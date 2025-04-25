package viewer

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
)

// Manager チャンネル閲覧者マネージャ
type Manager struct {
	hub      *hub.Hub
	channels map[uuid.UUID]map[*viewer]struct{}
	users    map[uuid.UUID]map[*viewer]struct{}
	viewers  map[interface{}]*viewer
	mu       sync.RWMutex
}

type viewer struct {
	key       interface{}
	connKey   string
	userID    uuid.UUID
	channelID uuid.UUID
	state     StateWithTime
}

// NewManager チャンネル閲覧者マネージャーを生成します
func NewManager(hub *hub.Hub) *Manager {
	vm := &Manager{
		hub:      hub,
		channels: map[uuid.UUID]map[*viewer]struct{}{},
		users:    map[uuid.UUID]map[*viewer]struct{}{},
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
func (vm *Manager) GetChannelViewers(channelID uuid.UUID) map[uuid.UUID]StateWithTime {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return calculateChannelViewers(vm.channels[channelID])
}

// SetViewer 指定したキーのチャンネル閲覧者状態を設定します
func (vm *Manager) SetViewer(key interface{}, connKey string, userID uuid.UUID, channelID uuid.UUID, state State) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	cv, ok := vm.channels[channelID]
	if !ok {
		cv = map[*viewer]struct{}{}
		vm.channels[channelID] = cv
	}
	uv, ok := vm.users[userID]
	if !ok {
		uv = map[*viewer]struct{}{}
		vm.users[userID] = uv
	}

	v, ok := vm.viewers[key]
	if ok {
		if v.channelID == channelID {
			if v.state.State == state {
				// 何も変わってない
				return
			}
			// stateだけ変更
			v.state.State = state
		} else {
			// channelとstateが変更
			oldC := v.channelID
			old := vm.channels[oldC]
			delete(old, v)

			v.channelID = channelID
			v.state = StateWithTime{
				State: state,
				Time:  time.Now(),
			}

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
			connKey:   connKey,
			userID:    userID,
			channelID: channelID,
			state: StateWithTime{
				State: state,
				Time:  time.Now(),
			},
		}
		vm.viewers[key] = v
	}

	cv[v] = struct{}{}
	uv[v] = struct{}{}
	vm.hub.Publish(hub.Message{
		Name: event.UserViewStateChanged,
		Fields: hub.Fields{
			"user_id":     userID,
			"view_states": calculateUserViewStates(uv),
		},
	})
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

	delete(vm.viewers, key)
	cv := vm.channels[v.channelID]
	delete(cv, v)
	uv := vm.users[v.userID]
	delete(uv, v)

	vm.hub.Publish(hub.Message{
		Name: event.UserViewStateChanged,
		Fields: hub.Fields{
			"user_id":     v.userID,
			"view_states": calculateUserViewStates(uv),
		},
	})
	vm.hub.Publish(hub.Message{
		Name: event.ChannelViewersChanged,
		Fields: hub.Fields{
			"channel_id": v.channelID,
			"viewers":    calculateChannelViewers(cv),
		},
	})
}

// 5分に1回呼び出される。チャンネルマップとユーザーマップのお掃除
func (vm *Manager) gc() {
	for cid, cv := range vm.channels {
		if len(cv) == 0 {
			delete(vm.channels, cid)
		}
	}
	for uid, uv := range vm.users {
		if len(uv) == 0 {
			delete(vm.users, uid)
		}
	}
}

// calculateUserViewStates ユーザーのviewerのsetからmap[conn_key]StateWithChannelを計算する
func calculateUserViewStates(uv map[*viewer]struct{}) map[string]StateWithChannel {
	result := make(map[string]StateWithChannel, len(uv))
	for v := range uv {
		result[v.connKey] = StateWithChannel{
			State:     v.state.State,
			ChannelID: v.channelID,
		}
	}
	return result
}

func calculateChannelViewers(vs map[*viewer]struct{}) map[uuid.UUID]StateWithTime {
	result := make(map[uuid.UUID]StateWithTime, len(vs))
	for v := range vs {
		if s, ok := result[v.userID]; ok && s.State > v.state.State {
			continue
		}
		result[v.userID] = v.state
	}
	return result
}
