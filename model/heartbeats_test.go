package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const testChannelID = "aaefc6cc-75e5-4eee-a2f3-cae63dc3ede8"

func TestGetHeartbeatStatus(t *testing.T) {
	assert := assert.New(t)

	statusesMutex.Lock()
	HeartbeatStatuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUserID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}
	statusesMutex.Unlock()

	status, ok := GetHeartbeatStatus(testChannelID)
	assert.Len(status.UserStatuses, 1)

	status, ok = GetHeartbeatStatus(CreateUUID())
	assert.False(ok)
}

func TestHeartbeat(t *testing.T) {
	assert := assert.New(t)

	tickTime = 10 * time.Millisecond
	timeoutDuration = -20 * time.Millisecond
	statusesMutex.Lock()
	HeartbeatStatuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUserID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	statusesMutex.Unlock()
	assert.Len(HeartbeatStatuses[testChannelID].UserStatuses, 1)

	require.NoError(t, HeartbeatStart())

	time.Sleep(50 * time.Millisecond)

	statusesMutex.Lock()
	assert.Len(HeartbeatStatuses[testChannelID].UserStatuses, 0)
	statusesMutex.Unlock()

	require.NoError(t, HeartbeatStop())
}
