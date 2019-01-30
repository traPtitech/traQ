package model

import (
	"github.com/satori/go.uuid"
	"testing"
	"time"
)

func TestGetHeartbeatStatus(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	statusesMutex.Lock()
	HeartbeatStatuses[channel.ID] = &HeartbeatStatus{
		ChannelID: channel.ID,
		UserStatuses: []*UserStatus{
			{
				UserID:   user.GetUID(),
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}
	statusesMutex.Unlock()

	status, _ := GetHeartbeatStatus(channel.ID)
	assert.Len(status.UserStatuses, 1)

	_, ok := GetHeartbeatStatus(uuid.NewV4())
	assert.False(ok)
}

func TestHeartbeat(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	tickTime = 10 * time.Millisecond
	timeoutDuration = 20 * time.Millisecond
	statusesMutex.Lock()
	HeartbeatStatuses[channel.ID] = &HeartbeatStatus{
		ChannelID: channel.ID,
		UserStatuses: []*UserStatus{
			{
				UserID:   user.GetUID(),
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	statusesMutex.Unlock()
	assert.Len(HeartbeatStatuses[channel.ID].UserStatuses, 1)

	require.NoError(HeartbeatStart())

	time.Sleep(50 * time.Millisecond)

	statusesMutex.Lock()
	assert.Nil(HeartbeatStatuses[channel.ID])
	statusesMutex.Unlock()

	require.NoError(HeartbeatStop())
}
