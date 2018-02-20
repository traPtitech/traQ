package model

import (
	"errors"
	"sync"
	"time"
)

// UserStatus userの状態
type UserStatus struct {
	UserID   string    `json:"userId"`
	Status   string    `json:"status"`
	LastTime time.Time `json:"-"`
}

// HeartbeatStatus Heartbeatの状態
type HeartbeatStatus struct {
	UserStatuses []*UserStatus `json:"userStatuses"`
	ChannelID    string        `json:"channelId"`
}

var (
	// HeartbeatStatuses HeartbeatStatusの全チャンネルのリスト
	HeartbeatStatuses = make(map[string]*HeartbeatStatus)

	ticker          *time.Ticker
	stop            chan bool
	tickerMutex     sync.Mutex
	statusesMutex   sync.RWMutex
	timeoutDuration = -5 * time.Second
	tickTime        = 500 * time.Millisecond
)

// UpdateHeartbeatStatuses UserIDで指定されたUserのHeartbeatの更新を行う
func UpdateHeartbeatStatuses(userID, channelID, status string) {
	statusesMutex.Lock()
	defer statusesMutex.Unlock()
	channelStatus, ok := HeartbeatStatuses[channelID]
	if !ok {
		HeartbeatStatuses[channelID] = &HeartbeatStatus{}
		channelStatus, _ = HeartbeatStatuses[channelID]
		channelStatus.ChannelID = channelID
	}

	for _, userStatus := range channelStatus.UserStatuses {
		if userStatus.UserID == userID {
			userStatus.LastTime = time.Now()
			userStatus.Status = status
			return
		}
	}
	userStatus := &UserStatus{
		UserID:   userID,
		Status:   status,
		LastTime: time.Now(),
	}
	channelStatus.UserStatuses = append(channelStatus.UserStatuses, userStatus)
}

func removeTimeoutStatus() {
	statusesMutex.Lock()
	defer statusesMutex.Unlock()
	removed := make(map[string]*HeartbeatStatus)
	for channelID, channelStatus := range HeartbeatStatuses {
		removed[channelID] = &HeartbeatStatus{}
		for _, userStatus := range channelStatus.UserStatuses {
			// 最終POSTから指定時間以上経ったものを削除する
			timeout := time.Now().Add(timeoutDuration)
			if timeout.Before(userStatus.LastTime) {
				removed[channelID].UserStatuses = append(removed[channelID].UserStatuses, userStatus)
			}
		}
	}
	HeartbeatStatuses = removed
}

// HeartbeatStart ハートビートをスタートする
func HeartbeatStart() error {
	stop = make(chan bool)
	if ticker != nil {
		return errors.New("Heartbeat already started")
	}
	tickerMutex.Lock()
	ticker = time.NewTicker(tickTime)
	go func() {
	loop:
		for {
			select {
			case <-ticker.C:
				removeTimeoutStatus()

			case <-stop:
				tickerMutex.Unlock()
				break loop
			}

		}
	}()
	return nil
}

// HeartbeatStop ハートビートをストップする
func HeartbeatStop() error {
	if stop == nil {
		return errors.New("HeartbeatStop before Start")
	}
	if ticker == nil {
		return errors.New("HeartbeatStop before Start")
	}

	close(stop)
	ticker.Stop()
	tickerMutex.Lock()
	stop = nil
	ticker = nil
	tickerMutex.Unlock()
	return nil
}

// GetHeartbeatStatus channelIDで指定したHeartbeatStatusを取得する
func GetHeartbeatStatus(channelID string) (HeartbeatStatus, bool) {
	statusesMutex.RLock()
	defer statusesMutex.RUnlock()
	status, ok := HeartbeatStatuses[channelID]
	if ok {
		return *status, ok
	}
	return HeartbeatStatus{}, ok
}
