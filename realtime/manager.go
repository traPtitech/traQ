package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
)

// Manager リアルタイム情報管理
type Manager struct {
	OnlineCounter *OnlineCounter
	ViewerManager *ViewerManager
	HeartBeats    *HeartBeats
}

// NewManager realtime.Managerを生成・起動します
func NewManager(hub *hub.Hub) *Manager {
	oc := newOnlineCounter(hub)
	hb := newHeartBeats(hub)
	vm := newViewerManager(hub, hb)

	go func() {
		for e := range hub.Subscribe(8, event.SSEConnected, event.SSEDisconnected).Receiver {
			switch e.Topic() {
			case event.SSEConnected:
				oc.Inc(e.Fields["user_id"].(uuid.UUID))
			case event.SSEDisconnected:
				oc.Dec(e.Fields["user_id"].(uuid.UUID))
			}
		}
	}()

	return &Manager{
		OnlineCounter: oc,
		ViewerManager: vm,
		HeartBeats:    hb,
	}
}
