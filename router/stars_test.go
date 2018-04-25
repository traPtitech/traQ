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
		var res []ChannelForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res)) {
			assert.Len(res, 1)
			assert.Equal(channel.ID, res[0].ChannelID)
		}
	}
}

func TestPutStars(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channelID := mustMakeChannel(t, testUser.ID, "test", true).ID

	c, rec := getContext(e, t, cookie, nil)
	c.SetParamNames("channelID")
	c.SetParamValues(channelID)
	requestWithContext(t, mw(PutStars), c)

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
