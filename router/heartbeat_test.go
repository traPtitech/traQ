package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

func TestPostHeartbeat(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	requestBody, err := json.Marshal(struct {
		ChannelID string `json:"channelId"`
		Status    string `json:"status"`
	}{
		ChannelID: testChannelID,
		Status:    "editing",
	})
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(requestBody))
	rec := request(e, t, mw(PostHeartbeat), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody HeartbeatStatus
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Equal(testChannelID, responseBody.ChannelID)
			assert.Len(responseBody.UserStatuses, 1)
			assert.Equal(testUser.ID, responseBody.UserStatuses[0].UserID)
			assert.Equal("editing", responseBody.UserStatuses[0].Status)
		}
	}
}

func TestGetHeartbeat(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	statuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUser.ID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	q := make(url.Values)
	q.Set("channelId", testChannelID)

	req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
	rec := request(e, t, mw(GetHeartbeat), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody HeartbeatStatus
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			t.Log(responseBody)
			assert.Equal(testChannelID, responseBody.ChannelID)
			assert.Len(responseBody.UserStatuses, 1)
			assert.Equal(testUser.ID, responseBody.UserStatuses[0].UserID)
			assert.Equal("editing", responseBody.UserStatuses[0].Status)
		}
	}
}

func TestGetHeartbeatStatus(t *testing.T) {
	statusesMutex.Lock()
	statuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUser.ID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}
	statusesMutex.Unlock()

	status, ok := GetHeartbeatStatus(testChannelID)
	if assert.True(t, ok) {
		assert.Len(t, status.UserStatuses, 1)
	}
	status, ok = GetHeartbeatStatus(model.CreateUUID())
	assert.False(t, ok)
}

func TestHeartbeat(t *testing.T) {
	tickTime = 10 * time.Millisecond
	timeoutDuration = -20 * time.Millisecond
	statusesMutex.Lock()
	statuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUser.ID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	statusesMutex.Unlock()
	require.Len(t, statuses[testChannelID].UserStatuses, 1)
	require.NoError(t, HeartbeatStart())

	time.Sleep(50 * time.Millisecond)

	statusesMutex.Lock()
	assert.Len(t, statuses[testChannelID].UserStatuses, 0)
	statusesMutex.Unlock()

	require.NoError(t, HeartbeatStop())
}
