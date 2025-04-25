package viewer

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/utils/set"
	"github.com/traPtitech/traQ/utils/throttle"
)

const throttleInterval = 3 * time.Second

// if the number of viewers is greater than the threshold, throttle the event
// otherwise, publish the event immediately
const throttleThreshold = 30

// Manager チャンネル閲覧者マネージャ
type Manager struct {
	hub             *hub.Hub
	channels        map[uuid.UUID]*set.Set[*viewer]
	users           map[uuid.UUID]*set.Set[*viewer]
	viewers         map[any]*viewer
	channelThrottle *throttle.Map[uuid.UUID]
	userThrottle    *throttle.Map[uuid.UUID]
	mu              sync.RWMutex
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
		channels: make(map[uuid.UUID]*set.Set[*viewer]),
		users:    make(map[uuid.UUID]*set.Set[*viewer]),
		viewers:  make(map[any]*viewer),
	}
	vm.channelThrottle = throttle.NewThrottleMap(throttleInterval, 5*time.Minute, func(cid uuid.UUID) {
		vm.mu.Lock()
		defer vm.mu.Unlock()
		vm.publishChannelChanged(cid)
	})
	vm.userThrottle = throttle.NewThrottleMap(throttleInterval, 5*time.Minute, func(uid uuid.UUID) {
		vm.mu.Lock()
		defer vm.mu.Unlock()
		vm.publishUserViewStateChanged(uid)
	})

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

	cv, exists := vm.channels[channelID]
	if !exists {
		cv = set.New[*viewer]()
		vm.channels[channelID] = cv
	}
	uv, exists := vm.users[userID]
	if !exists {
		uv = set.New[*viewer]()
		vm.users[userID] = uv
	}

	v, ok := vm.viewers[key]
	if ok {
		if v.channelID == channelID {
			if v.state.State == state {
				return // nothing changed
			}
			v.state.State = state
		} else {
			// channelとstateが変更
			previousChannelID := v.channelID
			viewers := vm.channels[previousChannelID]
			viewers.Remove(v)

			v.channelID = channelID
			v.state = StateWithTime{
				State: state,
				Time:  time.Now(),
			}

			vm.notifyChannelViewers(previousChannelID)
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

	cv.Add(v)
	uv.Add(v)

	vm.notifyChannelViewers(channelID)
	vm.notifyUserViewStateChanged(userID)
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
	cv.Remove(v)
	uv := vm.users[v.userID]
	uv.Remove(v)

	vm.notifyChannelViewers(v.channelID)
	vm.notifyUserViewStateChanged(v.userID)
}

// 5分に1回呼び出される。チャンネルマップとユーザーマップのお掃除
func (vm *Manager) gc() {
	for cid, cv := range vm.channels {
		if cv.Len() == 0 {
			delete(vm.channels, cid)
		}
	}
	for uid, uv := range vm.users {
		if uv.Len() == 0 {
			delete(vm.users, uid)
		}
	}
}

func (vm *Manager) notifyChannelViewers(channelID uuid.UUID) {
	cv := vm.channels[channelID]
	if cv.Len() > throttleThreshold {
		vm.channelThrottle.Trigger(channelID)
	} else {
		vm.channelThrottle.Stop(channelID)
		vm.publishChannelChanged(channelID)
	}
}

func (vm *Manager) notifyUserViewStateChanged(userID uuid.UUID) {
	uv := vm.users[userID]
	if uv.Len() > throttleThreshold {
		vm.userThrottle.Trigger(userID)
	} else {
		vm.userThrottle.Stop(userID)
		vm.publishUserViewStateChanged(userID)
	}
}

func (vm *Manager) publishChannelChanged(channelID uuid.UUID) {
	channelViewers := vm.channels[channelID]
	vm.hub.Publish(hub.Message{
		Name: event.ChannelViewersChanged,
		Fields: hub.Fields{
			"channel_id": channelID,
			"viewers":    calculateChannelViewers(channelViewers),
		},
	})
}

func (vm *Manager) publishUserViewStateChanged(userID uuid.UUID) {
	userViewers := vm.users[userID]
	vm.hub.Publish(hub.Message{
		Name: event.UserViewStateChanged,
		Fields: hub.Fields{
			"user_id":     userID,
			"view_states": calculateUserViewStates(userViewers),
		},
	})
}

// calculateUserViewStates ユーザーのviewerのsetからmap[conn_key]StateWithChannelを計算する
func calculateUserViewStates(uv *set.Set[*viewer]) map[string]StateWithChannel {
	result := make(map[string]StateWithChannel, uv.Len())
	for v := range uv.Values() {
		result[v.connKey] = StateWithChannel{
			State:     v.state.State,
			ChannelID: v.channelID,
		}
	}
	return result
}

func calculateChannelViewers(vs *set.Set[*viewer]) map[uuid.UUID]StateWithTime {
	result := make(map[uuid.UUID]StateWithTime, vs.Len())
	for v := range vs.Values() {
		if s, ok := result[v.userID]; ok && s.State > v.state.State {
			continue
		}
		result[v.userID] = v.state
	}
	return result
}
