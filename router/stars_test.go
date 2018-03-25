package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStars(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	mustStarChannel(t, testUser.ID, channel.ID)

	rec := request(e, t, mw(GetStars), cookie, nil)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody []ChannelForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 1)
			assert.Equal(channel.ID, responseBody[0].ChannelID)
		}
	}
}

func TestPostStars(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)

	post := struct {
		ChannelID string `json:"channelId"`
	}{
		ChannelID: channel.ID,
	}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostStars), cookie, req)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}

func TestDeleteStars(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	mustStarChannel(t, testUser.ID, channel.ID)

	post := struct {
		ChannelID string `json:"channelID"`
	}{
		ChannelID: channel.ID,
	}

	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeleteStars), cookie, req)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}
