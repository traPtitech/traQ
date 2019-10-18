package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"sync"
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
	return stringViewStates[s]
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
	UserID    uuid.UUID
	ChannelID uuid.UUID
	State     ViewState
}

// ViewerManager チャンネルビュアーマネージャ
type ViewerManager struct {
	hub      *hub.Hub
	hb       *HeartBeats
	channels map[uuid.UUID]map[*viewer]struct{}
	mu       sync.RWMutex
}

func newViewerManager(hub *hub.Hub, hb *HeartBeats) *ViewerManager {
	return &ViewerManager{
		hub:      hub,
		hb:       hb,
		channels: map[uuid.UUID]map[*viewer]struct{}{},
	}
}

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
			if s, ok := result[v.UserID]; ok {
				if s > v.State {
					continue
				}
			}
			result[v.UserID] = v.State
		}
	}
	vm.mu.RUnlock()

	return result
}
