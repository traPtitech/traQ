package realtime

import "github.com/leandro-lugaresi/hub"

// Manager リアルタイム情報管理
type Manager struct {
	OnlineCounter *OnlineCounter
	ViewerManager *ViewerManager
	HeartBeats    *HeartBeats
}

// NewManager realtime.Managerを生成・起動します
func NewManager(hub *hub.Hub) *Manager {
	oc := newOnlineCounter(hub)
	hb := newHeartBeats(hub, oc)
	vm := newViewerManager(hub, hb)

	return &Manager{
		OnlineCounter: oc,
		ViewerManager: vm,
		HeartBeats:    hb,
	}
}
