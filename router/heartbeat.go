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

// HeartbeatStatuses Heartbeatの状態
type HeartbeatStatuses struct {
	UserStatuses []*UserStatus `json:"userStatuses"`
	ChannelID    string        `json:"channelId"`
}

var (
	t             *time.Ticker
	stop          chan bool
	statusesMutex sync.RWMutex
	statuses      = make(map[string]*HeartbeatStatuses)
	lastUpdated   time.Time
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
		statuses[channelID] = &HeartbeatStatuses{}
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
	removed := make(map[string]*HeartbeatStatuses)
	for channelID, channelStatus := range statuses {
		removed[channelID] = &HeartbeatStatuses{}
		for _, userStatus := range channelStatus.UserStatuses {
			// 最終POSTから5秒以上経ったものを削除する
			timeout := time.Now().Add(-5 * time.Second)
			if timeout.After(userStatus.LastTime) {
				removed[channelID].UserStatuses = append(removed[channelID].UserStatuses, userStatus)
			}
		}
	}
	statuses = removed
}

// HeartbeatStart ハートビートをスタートする
func HeartbeatStart() error {
	stop = make(chan bool)
	if t != nil {
		return errors.New("Heartbeat already started")
	}
	t = time.NewTicker(500 * time.Millisecond)
	go func() {
	loop:
		for {
			select {
			case <-t.C:
				removeTimeoutStatus()

			case <-stop:
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
	if t == nil {
		return errors.New("HeartbeatStop before Start")
	}

	close(stop)
	t.Stop()
	stop = nil
	t = nil
	return nil
}
