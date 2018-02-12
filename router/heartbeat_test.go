package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/traPtitech/traQ/model"
)

func TestPostHeartbeat(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	requestBody, err := json.Marshal(struct {
		ChannelID string `json:"channelId"`
		Status    string `json:"status"`
	}{
		ChannelID: testChannelID,
		Status:    "editing",
	})

	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(requestBody))
	rec := request(e, t, mw(PostHeartbeat), cookie, req)

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody model.HeartbeatStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if responseBody.ChannelID != testChannelID {
		t.Fatalf("ChannelID wrong: want %s, actual %s", testChannelID, responseBody.ChannelID)
	}

	if len(responseBody.UserStatuses) != 1 {
		t.Fatalf("UserStatuses length wrong: want 1, actual %d", len(responseBody.UserStatuses))
	}

	if responseBody.UserStatuses[0].UserID != testUser.ID {
		t.Fatalf("ChannelID wrong: want %s, actual %s", testUser.ID, responseBody.UserStatuses[0].UserID)
	}

	if responseBody.UserStatuses[0].Status != "editing" {
		t.Fatalf("ChannelID wrong: want editing, actual %s", responseBody.UserStatuses[0].Status)
	}
}

func TestGetHeartbeat(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	model.HeartbeatStatuses[testChannelID] = &model.HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*model.UserStatus{
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

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody model.HeartbeatStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}
	t.Log(responseBody)

	if responseBody.ChannelID != testChannelID {
		t.Fatalf("ChannelID wrong: want %s, actual %s", testChannelID, responseBody.ChannelID)
	}

	if len(responseBody.UserStatuses) != 1 {
		t.Fatalf("UserStatuses length wrong: want 1, actual %d", len(responseBody.UserStatuses))
	}

	if responseBody.UserStatuses[0].UserID != testUser.ID {
		t.Fatalf("ChannelID wrong: want %s, actual %s", testUser.ID, responseBody.UserStatuses[0].UserID)
	}

	if responseBody.UserStatuses[0].Status != "editing" {
		t.Fatalf("ChannelID wrong: want editing, actual %s", responseBody.UserStatuses[0].Status)
	}
}
