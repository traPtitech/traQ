package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/traPtitech/traQ/model"
	"net/http"
)

func TestPostHeartbeat(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "testChan", "", true)

	requestBody, err := json.Marshal(struct {
		ChannelID string `json:"channelId"`
		Status    string `json:"status"`
	}{
		ChannelID: channel.ID,
		Status:    "editing",
	})
	require.NoError(err)
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(requestBody))
	rec := request(e, t, mw(PostHeartbeat), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody model.HeartbeatStatus
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Equal(channel.ID, responseBody.ChannelID)
			assert.Len(responseBody.UserStatuses, 1)
			assert.Equal(testUser.ID, responseBody.UserStatuses[0].UserID)
			assert.Equal("editing", responseBody.UserStatuses[0].Status)
		}
	}
}

func TestGetHeartbeat(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "testChan", "", true)

	model.HeartbeatStatuses[channel.ID] = &model.HeartbeatStatus{
		ChannelID: channel.ID,
		UserStatuses: []*model.UserStatus{
			{
				UserID:   testUser.ID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	q := make(url.Values)
	q.Set("channelId", channel.ID)

	req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)
	rec := request(e, t, mw(GetHeartbeat), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody model.HeartbeatStatus
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Equal(channel.ID, responseBody.ChannelID)
			assert.Len(responseBody.UserStatuses, 1)
			assert.Equal(testUser.ID, responseBody.UserStatuses[0].UserID)
			assert.Equal("editing", responseBody.UserStatuses[0].Status)
		}
	}
}
