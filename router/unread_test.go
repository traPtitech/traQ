package router

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetUnread(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")
	testMessage := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	// 正常系
	mustMakeUnread(t, testUser.GetUID(), testMessage.GetID())
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/me/unread")
	requestWithContext(t, mw(GetUnread), c)

	assert.EqualValues(http.StatusOK, rec.Code)
	var responseBody []*MessageForResponse
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody))
	assert.Len(responseBody, 1)
}

func TestDeleteUnread(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")
	testMessage := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	// 正常系
	mustMakeUnread(t, testUser.GetUID(), testMessage.GetID())

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/me/unread/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteUnread), c)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}
