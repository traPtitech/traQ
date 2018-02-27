package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUnread(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	testMessage := mustMakeMessage(t, testUser.ID, channel.ID)

	// 正常系
	mustMakeUnread(t, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/me/unread")
	requestWithContext(t, mw(GetUnread), c)

	assert.EqualValues(http.StatusOK, rec.Code)
	var responseBody []*MessageForResponse
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody))
	assert.Len(responseBody, 1)
	correctResponse := formatMessage(testMessage)
	correctResponse.Datetime = parseDateTime(correctResponse.Datetime)
	assert.Equal(*responseBody[0], *correctResponse)
}

func TestDeleteUnread(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	testMessage := mustMakeMessage(t, testUser.ID, channel.ID)

	// 正常系
	mustMakeUnread(t, testUser.ID, testMessage.ID)
	post := []string{testMessage.ID}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/users/me/unread")
	requestWithContext(t, mw(DeleteUnread), c)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}
