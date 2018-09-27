package router

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetStars(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")
	mustStarChannel(t, testUser.GetUID(), channel.GetCID())

	rec := request(e, t, mw(GetStars), cookie, nil)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var res []string
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res)) {
			assert.Len(res, 1)
			assert.Equal(channel.ID, res[0])
		}
	}
}

func TestPutStars(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")

	c, rec := getContext(e, t, cookie, nil)
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutStars), c)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}

func TestDeleteStars(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")
	mustStarChannel(t, testUser.GetUID(), channel.GetCID())

	c, rec := getContext(e, t, cookie, nil)
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteStars), c)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}
