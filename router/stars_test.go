package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGetStars(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, err := makeChannel(testUser.ID, "test", true)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if err := starChannel(testUser.ID, channel.ID); err != nil {
		t.Fatalf("failed to star message: %v", err)
	}

	rec := request(e, t, mw(GetStars), cookie, nil)
	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []ChannelForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response channels length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].ChannelID != channel.ID {
		t.Fatalf("channel ID is wrong: want %v, actual %v", channel.ID, responseBody[0].ChannelID)
	}
}

func TestPostStars(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, err := makeChannel(testUser.ID, "test", true)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	post := struct {
		ChannelID string `json:"channelId"`
	}{
		ChannelID: channel.ID,
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostStars), cookie, req)

	if rec.Code != 201 {
		t.Fatalf("Response code wrong: want 201, actual %d", rec.Code)
	}

	var responseBody []ChannelForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response Channels length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].ChannelID != channel.ID {
		t.Fatalf("message text is wrong: want %v, actual %v", channel.ID, responseBody[0].ChannelID)
	}
}

func TestDeleteStars(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, err := makeChannel(testUser.ID, "test", true)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	if err := starChannel(testUser.ID, channel.ID); err != nil {
		t.Fatalf("failed to star message: %v", err)
	}

	post := struct {
		ChannelID string `json:"channelID"`
	}{
		ChannelID: channel.ID,
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeleteStars), cookie, req)

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []ChannelForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 0 {
		t.Fatalf("Response Channels length wrong: want 1, actual %d", len(responseBody))
	}
}
