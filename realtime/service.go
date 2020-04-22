package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/realtime/webrtc"
	"github.com/traPtitech/traQ/realtime/webrtcv3"
)

// Service リアルタイム情報管理
type Service struct {
	OnlineCounter *OnlineCounter
	ViewerManager *viewer.Manager
	HeartBeats    *HeartBeats
	WebRTC        *webrtc.Manager
	WebRTCv3      *webrtcv3.Manager
}

// NewService realtime.Serviceを生成・起動します
func NewService(hub *hub.Hub) *Service {
	oc := newOnlineCounter(hub)
	vm := viewer.NewManager(hub)
	hb := newHeartBeats(vm)
	wr := webrtc.NewManager(hub)
	wrv3 := webrtcv3.NewManager(hub)

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
		WebRTC:        wr,
		WebRTCv3:      wrv3,
	}
}
