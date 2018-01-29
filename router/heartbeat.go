package router

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
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
	ticker          *time.Ticker
	stop            chan bool
	tickerMutex     sync.Mutex
	statusesMutex   sync.RWMutex
	statuses        = make(map[string]*HeartbeatStatus)
	timeoutDuration = -5 * time.Second
	tickTime        = 500 * time.Millisecond
)

// PostHeartbeat POST /heartbeat のハンドラ
func PostHeartbeat(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	requestBody := struct {
		ChannelID string `json:"channelId"`
		Status    string `json:"status"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}
	updateStatuses(userID, requestBody.ChannelID, requestBody.Status)

	statusesMutex.RLock()
	defer statusesMutex.RUnlock()
	return c.JSON(http.StatusOK, statuses[requestBody.ChannelID])
}

// GetHeartbeat GET /heartbeat のハンドラ
func GetHeartbeat(c echo.Context) error {
	requestBody := struct {
		ChannelID string `query:"channelId"`
	}{}
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request query")
	}

	statusesMutex.RLock()
	defer statusesMutex.RUnlock()
	return c.JSON(http.StatusOK, statuses[requestBody.ChannelID])
}

func updateStatuses(userID, channelID, status string) {
	statusesMutex.Lock()
	defer statusesMutex.Unlock()
	channelStatus, ok := statuses[channelID]
	if !ok {
		statuses[channelID] = &HeartbeatStatus{}
		channelStatus, _ = statuses[channelID]
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
	for channelID, channelStatus := range statuses {
		removed[channelID] = &HeartbeatStatus{}
		for _, userStatus := range channelStatus.UserStatuses {
			// 最終POSTから指定時間以上経ったものを削除する
			timeout := time.Now().Add(timeoutDuration)
			if timeout.Before(userStatus.LastTime) {
				removed[channelID].UserStatuses = append(removed[channelID].UserStatuses, userStatus)
			}
		}
	}
	statuses = removed
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
	status, ok := statuses[channelID]
	if ok {
		return *status, ok
	}
	return HeartbeatStatus{}, ok
}
