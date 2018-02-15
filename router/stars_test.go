package router

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStars(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
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
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)

	post := struct {
		ChannelID string `json:"channelId"`
	}{
		ChannelID: channel.ID,
	}
	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostStars), cookie, req)

	if assert.EqualValues(http.StatusCreated, rec.Code) {
		var responseBody []ChannelForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 1)
			assert.Equal(channel.ID, responseBody[0].ChannelID)
		}
	}
}

func TestDeleteStars(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	mustStarChannel(t, testUser.ID, channel.ID)

	post := struct {
		ChannelID string `json:"channelID"`
	}{
		ChannelID: channel.ID,
	}

	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeleteStars), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody []ChannelForResponse

		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 0)
		}
	}
}
