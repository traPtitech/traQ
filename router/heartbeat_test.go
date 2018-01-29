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

	var responseBody HeartbeatStatus
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

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody HeartbeatStatus
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

	if len(status.UserStatuses) != 1 {
		t.Fatalf("statuses length wrong: want 1, actual %d", len(status.UserStatuses))
	}

	status, ok = GetHeartbeatStatus(model.CreateUUID())

	if ok {
		t.Fatalf("ok is not false")
	}
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
	if len(statuses[testChannelID].UserStatuses) != 1 {
		t.Fatalf("statuses length wrong: want 1, actual %d", len(statuses[testChannelID].UserStatuses))
	}

	if err := HeartbeatStart(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	statusesMutex.Lock()
	if len(statuses[testChannelID].UserStatuses) != 0 {
		t.Fatalf("statuses length wrong: want 0, actual %d", len(statuses[testChannelID].UserStatuses))
	}
	statusesMutex.Unlock()

	if err := HeartbeatStop(); err != nil {
		t.Fatal(err)
	}

}
