package viewer

import (
	"iter"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
)

// Manager チャンネル閲覧者マネージャ
type Manager struct {
	hub      *hub.Hub
	channels map[uuid.UUID]*viewerSet
	users    map[uuid.UUID]*viewerSet
	viewers  map[any]*viewer
	mu       sync.RWMutex
}

type viewer struct {
	key       any
	connKey   string
	userID    uuid.UUID
	channelID uuid.UUID
	state     StateWithTime
}

// NewManager チャンネル閲覧者マネージャーを生成します
func NewManager(hub *hub.Hub) *Manager {
	vm := &Manager{
		hub:      hub,
		channels: make(map[uuid.UUID]*viewerSet),
		users:    make(map[uuid.UUID]*viewerSet),
		viewers:  make(map[any]*viewer),
	}

	// start garbage collector
	go func() {
		for range time.Tick(5 * time.Minute) {
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
func (vm *Manager) SetViewer(key any, connKey string, userID uuid.UUID, channelID uuid.UUID, state State) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	cv := vm.ensureChannelViewers(channelID)
	uv := vm.ensureUserViewers(userID)

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
			previousChannelID := v.channelID
			viewers := vm.channels[previousChannelID]
			viewers.remove(v)

			v.channelID = channelID
			v.state = StateWithTime{
				State: state,
				Time:  time.Now(),
			}

			vm.hub.Publish(hub.Message{
				Name: event.ChannelViewersChanged,
				Fields: hub.Fields{
					"channel_id": previousChannelID,
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

	cv.add(v)
	uv.add(v)
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
func (vm *Manager) RemoveViewer(key any) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	v, ok := vm.viewers[key]
	if !ok {
		return
	}

	delete(vm.viewers, key)
	cv := vm.channels[v.channelID]
	cv.remove(v)
	uv := vm.users[v.userID]
	uv.remove(v)

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
		if cv.len() == 0 {
			delete(vm.channels, cid)
		}
	}
	for uid, uv := range vm.users {
		if uv.len() == 0 {
			delete(vm.users, uid)
		}
	}
}

func (vm *Manager) ensureChannelViewers(channelID uuid.UUID) *viewerSet {
	cv, exists := vm.channels[channelID]
	if !exists {
		cv = newViewerSet()
		vm.channels[channelID] = cv
	}
	return cv
}

func (vm *Manager) ensureUserViewers(userID uuid.UUID) *viewerSet {
	uv, exists := vm.users[userID]
	if !exists {
		uv = newViewerSet()
		vm.users[userID] = uv
	}
	return uv
}

// calculateUserViewStates ユーザーのviewerのsetからmap[conn_key]StateWithChannelを計算する
func calculateUserViewStates(uv *viewerSet) map[string]StateWithChannel {
	result := make(map[string]StateWithChannel, uv.len())
	for v := range uv.values() {
		result[v.connKey] = StateWithChannel{
			State:     v.state.State,
			ChannelID: v.channelID,
		}
	}
	return result
}

func calculateChannelViewers(vs *viewerSet) map[uuid.UUID]StateWithTime {
	result := make(map[uuid.UUID]StateWithTime, vs.len())
	for v := range vs.values() {
		if s, ok := result[v.userID]; ok && s.State > v.state.State {
			continue
		}
		result[v.userID] = v.state
	}
	return result
}

type viewerSet struct {
	set map[*viewer]struct{}
}

func newViewerSet() *viewerSet {
	return &viewerSet{
		set: make(map[*viewer]struct{}),
	}
}

func (vs *viewerSet) add(v *viewer) {
	vs.set[v] = struct{}{}
}

func (vs *viewerSet) remove(v *viewer) {
	delete(vs.set, v)
}

func (vs *viewerSet) len() int {
	return len(vs.set)
}

func (vs *viewerSet) values() iter.Seq[*viewer] {
	return func(yield func(*viewer) bool) {
		for v := range vs.set {
			if !yield(v) {
				return
			}
		}
	}
}
