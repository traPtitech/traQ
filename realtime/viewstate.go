package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"strings"
	"sync"
	"time"
)

// ViewState 閲覧状態
type ViewState int

const (
	// ViewStateNone バックグランド表示中
	ViewStateNone ViewState = iota
	// ViewStateMonitoring メッセージ表示中
	ViewStateMonitoring
	// ViewStateEditing メッセージ入力中
	ViewStateEditing
)

// String string表記にします
func (s ViewState) String() string {
	return viewStateStrings[s]
}

// FromString stringからViewStateに変換します
func FromString(s string) ViewState {
	return stringViewStates[strings.ToLower(s)]
}

var viewStateStrings = map[ViewState]string{
	ViewStateNone:       "none",
	ViewStateEditing:    "editing",
	ViewStateMonitoring: "monitoring",
}

var stringViewStates map[string]ViewState

func init() {
	stringViewStates = map[string]ViewState{}
	for v, k := range viewStateStrings {
		stringViewStates[k] = v
	}
}

type viewer struct {
	key       interface{}
	userID    uuid.UUID
	channelID uuid.UUID
	state     ViewState
}

// ViewerManager チャンネルビュアーマネージャ
type ViewerManager struct {
	hub      *hub.Hub
	hb       *HeartBeats
	channels map[uuid.UUID]map[*viewer]struct{}
	viewers  map[interface{}]*viewer
	mu       sync.RWMutex
}

func newViewerManager(hub *hub.Hub, hb *HeartBeats) *ViewerManager {
	vm := &ViewerManager{
		hub:      hub,
		hb:       hb,
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
func (vm *ViewerManager) GetChannelViewers(channelID uuid.UUID) map[uuid.UUID]ViewState {
	result := map[uuid.UUID]ViewState{}

	hs, ok := vm.hb.GetHearts(channelID)
	if ok {
		for _, v := range hs.UserStatuses {
			result[v.UserID] = FromString(v.Status)
		}
	}

	vm.mu.RLock()
	vs, ok := vm.channels[channelID]
	if ok {
		for v := range vs {
			if s, ok := result[v.userID]; ok {
				if s > v.state {
					continue
				}
			}
			result[v.userID] = v.state
		}
	}
	vm.mu.RUnlock()

	return result
}

// SetViewer 指定したキーのチャンネル閲覧者状態を設定します
func (vm *ViewerManager) SetViewer(key interface{}, userID uuid.UUID, channelID uuid.UUID, state ViewState) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	v, ok := vm.viewers[key]
	if ok {
		// 前の状態を削除
		delete(vm.channels[v.channelID], v)
	} else {
		v = &viewer{
			key:       key,
			userID:    userID,
			channelID: channelID,
			state:     state,
		}
		vm.viewers[key] = v
	}
	v.channelID = channelID
	v.userID = userID
	v.state = state

	cv, ok := vm.channels[channelID]
	if !ok {
		cv = map[*viewer]struct{}{}
		vm.channels[channelID] = cv
	}
	cv[v] = struct{}{}
}

// RemoveViewer 指定したキーのチャンネル閲覧者状態を削除します
func (vm *ViewerManager) RemoveViewer(key interface{}) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	v, ok := vm.viewers[key]
	if !ok {
		return
	}
	delete(vm.viewers, key)
	delete(vm.channels[v.channelID], v)
}

// 5分に１回呼び出される。チャンネルマップのお掃除
func (vm *ViewerManager) gc() {
	for cid, cv := range vm.channels {
		if len(cv) == 0 {
			delete(vm.channels, cid)
		}
	}
}
