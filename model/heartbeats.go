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

type userOnlineStatus struct {
	sync.RWMutex
	id      string
	counter int
	time    time.Time
}

func (s *userOnlineStatus) isOnline() (r bool) {
	s.RLock()
	r = s.counter > 0
	s.RUnlock()
	return
}

func (s *userOnlineStatus) inc() {
	s.Lock()
	s.counter++
	if s.counter == 1 && OnUserOnlineStateChanged != nil {
		OnUserOnlineStateChanged(s.id, true)
	}
	s.Unlock()
}

func (s *userOnlineStatus) dec() {
	s.Lock()
	if s.counter > 0 {
		s.counter--
		if s.counter == 0 && OnUserOnlineStateChanged != nil {
			OnUserOnlineStateChanged(s.id, false)
		}
	}
	s.Unlock()
}

func (s *userOnlineStatus) setTime(time time.Time) {
	s.Lock()
	s.time = time
	s.Unlock()
}

func (s *userOnlineStatus) getTime() (t time.Time) {
	s.RLock()
	t = s.time
	s.RUnlock()
	return
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

	currentUserOnlineMap sync.Map

	// OnUserOnlineStateChanged ユーザーのオンライン状況が変化した時のイベントハンドラ
	OnUserOnlineStateChanged func(id string, online bool)
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

	t := time.Now()
	s, _ := currentUserOnlineMap.LoadOrStore(userID, &userOnlineStatus{id: userID})
	s.(*userOnlineStatus).setTime(t)
	for _, userStatus := range channelStatus.UserStatuses {
		if userStatus.UserID == userID {
			userStatus.LastTime = t
			userStatus.Status = status
			return
		}
	}
	userStatus := &UserStatus{
		UserID:   userID,
		Status:   status,
		LastTime: t,
	}
	channelStatus.UserStatuses = append(channelStatus.UserStatuses, userStatus)
	s.(*userOnlineStatus).inc()
}

func removeTimeoutStatus() {
	statusesMutex.Lock()
	defer statusesMutex.Unlock()
	removed := make(map[string]*HeartbeatStatus)
	timeout := time.Now().Add(timeoutDuration)
	for channelID, channelStatus := range HeartbeatStatuses {
		removed[channelID] = &HeartbeatStatus{}
		for _, userStatus := range channelStatus.UserStatuses {
			// 最終POSTから指定時間以上経ったものを削除する
			if timeout.Before(userStatus.LastTime) {
				removed[channelID].UserStatuses = append(removed[channelID].UserStatuses, userStatus)
				s, ok := currentUserOnlineMap.Load(userStatus.UserID)
				if ok {
					s.(*userOnlineStatus).dec()
					go UpdateUserLastOnline(userStatus.UserID, s.(*userOnlineStatus).getTime()) //DBに反映するのはこの時点
				}
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

// IsUserOnline ユーザーがオンラインかどうかを返します。
func IsUserOnline(userID string) bool {
	s, ok := currentUserOnlineMap.Load(userID)
	if !ok {
		return false
	}
	return s.(*userOnlineStatus).isOnline()
}
