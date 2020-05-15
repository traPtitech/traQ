package heartbeat

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/service/viewer"
	"sync"
	"time"
)

const (
	timeoutDuration = 5 * time.Second
	tickTime        = 500 * time.Millisecond
)

// Manager ハートビートマネージャー
type Manager struct {
	vm           *viewer.Manager
	channelBeats map[uuid.UUID][]*beat
	mu           sync.RWMutex
}

type beat struct {
	userID   uuid.UUID
	lastTime time.Time
}

// NewManager ハートビートマネージャーを生成します
func NewManager(vm *viewer.Manager) *Manager {
	h := &Manager{
		vm:           vm,
		channelBeats: make(map[uuid.UUID][]*beat),
	}
	go func() {
		t := time.NewTicker(tickTime)
		for range t.C {
			h.onTick()
		}
	}()
	return h
}

func (h *Manager) onTick() {
	h.mu.Lock()
	defer h.mu.Unlock()
	timeout := time.Now().Add(-1 * timeoutDuration)
	updated := make(map[uuid.UUID][]*beat)
	for cid, beats := range h.channelBeats {
		arr := make([]*beat, 0, len(beats))
		for _, b := range beats {
			// 最終POSTから指定時間以上経ったものを削除する
			if timeout.Before(b.lastTime) {
				arr = append(arr, b)
			} else {
				h.vm.RemoveViewer(b)
			}
		}
		if len(arr) > 0 {
			updated[cid] = arr
		}
	}
	h.channelBeats = updated
}

// Beat ハートビートを打ちます
func (h *Manager) Beat(userID, channelID uuid.UUID, status string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	beats, ok := h.channelBeats[channelID]
	if !ok {
		beats = make([]*beat, 0)
		h.channelBeats[channelID] = beats
	}

	t := time.Now()
	for _, b := range beats {
		if b.userID == userID {
			b.lastTime = t
			h.vm.SetViewer(b, userID, channelID, viewer.StateFromString(status))
			return
		}
	}
	b := &beat{
		userID:   userID,
		lastTime: t,
	}
	h.channelBeats[channelID] = append(beats, b)
	h.vm.SetViewer(b, userID, channelID, viewer.StateFromString(status))
}
