package realtime

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"sync"
	"time"
)

var (
	timeoutDuration = 5 * time.Second
	tickTime        = 500 * time.Millisecond
)

// HeartBeats ハートビートマネージャー
type HeartBeats struct {
	hub               *hub.Hub
	heartbeatStatuses map[uuid.UUID]*model.HeartbeatStatus
	sync.RWMutex
}

func newHeartBeats(hub *hub.Hub) *HeartBeats {
	h := &HeartBeats{
		hub:               hub,
		heartbeatStatuses: make(map[uuid.UUID]*model.HeartbeatStatus),
	}
	go func() {
		t := time.NewTicker(tickTime)
		for range t.C {
			h.onTick()
		}
	}()
	return h
}

func (h *HeartBeats) onTick() {
	h.Lock()
	defer h.Unlock()
	timeout := time.Now().Add(-1 * timeoutDuration)
	updated := make(map[uuid.UUID]*model.HeartbeatStatus)
	for cid, channelStatus := range h.heartbeatStatuses {
		arr := make([]*model.UserStatus, 0)
		for _, userStatus := range channelStatus.UserStatuses {
			// 最終POSTから指定時間以上経ったものを削除する
			if timeout.Before(userStatus.LastTime) {
				arr = append(arr, userStatus)
			}
		}
		if len(arr) > 0 {
			channelStatus.UserStatuses = arr
			updated[cid] = channelStatus
		}
	}
	h.heartbeatStatuses = updated
}

// Beat ハートビートを打ちます
func (h *HeartBeats) Beat(userID, channelID uuid.UUID, status string) {
	h.Lock()
	defer h.Unlock()
	channelStatus, ok := h.heartbeatStatuses[channelID]
	if !ok {
		channelStatus = &model.HeartbeatStatus{ChannelID: channelID}
		h.heartbeatStatuses[channelID] = channelStatus
	}

	t := time.Now()
	for _, userStatus := range channelStatus.UserStatuses {
		if userStatus.UserID == userID {
			userStatus.LastTime = t
			userStatus.Status = status
			return
		}
	}
	channelStatus.UserStatuses = append(channelStatus.UserStatuses, &model.UserStatus{
		UserID:   userID,
		Status:   status,
		LastTime: t,
	})
}

// GetHearts 指定したチャンネルのハートビートを取得します
func (h *HeartBeats) GetHearts(channelID uuid.UUID) (model.HeartbeatStatus, bool) {
	h.RLock()
	defer h.RUnlock()
	status, ok := h.heartbeatStatuses[channelID]
	if ok {
		return *status, ok
	}
	return model.HeartbeatStatus{}, ok
}
