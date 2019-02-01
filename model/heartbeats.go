package model

import (
	"errors"
	"github.com/satori/go.uuid"
	"sync"
	"time"
)

// UserStatus userの状態
type UserStatus struct {
	UserID   uuid.UUID `json:"userId"`
	Status   string    `json:"status"`
	LastTime time.Time `json:"-"`
}

// HeartbeatStatus Heartbeatの状態
type HeartbeatStatus struct {
	UserStatuses []*UserStatus `json:"userStatuses"`
	ChannelID    uuid.UUID     `json:"channelId"`
}

type userOnlineStatus struct {
	sync.RWMutex
	id      uuid.UUID
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
	HeartbeatStatuses = make(map[uuid.UUID]*HeartbeatStatus)

	ticker          *time.Ticker
	stop            chan bool
	tickerMutex     sync.Mutex
	statusesMutex   sync.RWMutex
	timeoutDuration = 5 * time.Second
	tickTime        = 500 * time.Millisecond

	currentUserOnlineMap sync.Map

	// OnUserOnlineStateChanged ユーザーのオンライン状況が変化した時のイベントハンドラ
	OnUserOnlineStateChanged func(id uuid.UUID, online bool)
)

// UpdateHeartbeatStatuses UserIDで指定されたUserのHeartbeatの更新を行う
func UpdateHeartbeatStatuses(userID, channelID uuid.UUID, status string) {
	statusesMutex.Lock()
	defer statusesMutex.Unlock()
	channelStatus, ok := HeartbeatStatuses[channelID]
	if !ok {
		channelStatus = &HeartbeatStatus{ChannelID: channelID}
		HeartbeatStatuses[channelID] = channelStatus
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
	timeout := time.Now().Add(-1 * timeoutDuration)
	updated := make(map[uuid.UUID]*HeartbeatStatus)
	for cid, channelStatus := range HeartbeatStatuses {
		arr := make([]*UserStatus, 0)
		for _, userStatus := range channelStatus.UserStatuses {
			if timeout.Before(userStatus.LastTime) {
				arr = append(arr, userStatus)
			} else {
				// 最終POSTから指定時間以上経ったものを削除する
				s, ok := currentUserOnlineMap.Load(userStatus.UserID)
				if ok {
					s.(*userOnlineStatus).dec()
					go UpdateUserLastOnline(userStatus.UserID, s.(*userOnlineStatus).getTime()) //nolint:errcheck
				}
			}
		}
		if len(arr) > 0 {
			channelStatus.UserStatuses = arr
			updated[cid] = channelStatus
		}
	}
	HeartbeatStatuses = updated
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
func GetHeartbeatStatus(channelID uuid.UUID) (HeartbeatStatus, bool) {
	statusesMutex.RLock()
	defer statusesMutex.RUnlock()
	status, ok := HeartbeatStatuses[channelID]
	if ok {
		return *status, ok
	}
	return HeartbeatStatus{}, ok
}

// IsUserOnline ユーザーがオンラインかどうかを返します。
func IsUserOnline(userID uuid.UUID) bool {
	s, ok := currentUserOnlineMap.Load(userID)
	if !ok {
		return false
	}
	return s.(*userOnlineStatus).isOnline()
}
