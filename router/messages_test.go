package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGetMessageByID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)

	requestWithContext(t, mw(GetMessageByID), c)

	assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String())
}

func TestGetMessagesByChannelID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)
	for i := 0; i < 5; i++ {
		mustMakeMessage(t, testUser.ID, channel.ID)
	}

	q := make(url.Values)
	q.Set("limit", "3")
	q.Set("offset", "1")
	req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/messages")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(GetMessagesByChannelID), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var responseBody []MessageForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 3)
		}
	}
}

func TestPostMessage(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)

	post := struct{ Text string }{Text: "test message"}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/messages")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PostMessage), c)

	if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
		message := &MessageForResponse{}
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), message)) {
			assert.Equal(post.Text, message.Content)
		}
	}
}

func TestPutMessageByID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)

	post := struct{ Text string }{Text: "test message"}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(PutMessageByID), c)

	message, err = model.GetMessageByID(message.ID)
	require.NoError(err)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		assert.Equal(post.Text, message.Text)
	}
}

func TestDeleteMessageByID(t *testing.T) {
	e, cookie, mw, _, require := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)

	req := httptest.NewRequest("DELETE", "http://test", nil)

	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(DeleteMessageByID), c)

	message, err := model.GetMessageByID(message.ID)
	require.Error(err)
}
