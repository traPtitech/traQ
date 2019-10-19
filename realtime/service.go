package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
)

// Service リアルタイム情報管理
type Service struct {
	OnlineCounter *OnlineCounter
	ViewerManager *ViewerManager
	HeartBeats    *HeartBeats
}

// NewService realtime.Serviceを生成・起動します
func NewService(hub *hub.Hub) *Service {
	oc := newOnlineCounter(hub)
	hb := newHeartBeats(hub)
	vm := newViewerManager(hub, hb)

	go func() {
		for e := range hub.Subscribe(8, event.SSEConnected, event.SSEDisconnected, event.WSConnected, event.WSDisconnected).Receiver {
			switch e.Topic() {
			case event.SSEConnected, event.WSConnected:
				oc.Inc(e.Fields["user_id"].(uuid.UUID))
			case event.SSEDisconnected, event.WSDisconnected:
				oc.Dec(e.Fields["user_id"].(uuid.UUID))
			}
		}
	}()

	return &Service{
		OnlineCounter: oc,
		ViewerManager: vm,
		HeartBeats:    hb,
	}
}
