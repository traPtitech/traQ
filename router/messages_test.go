package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
)

func TestGetMessageByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	message := mustMakeMessage(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)

	requestWithContext(t, mw(GetMessageByID), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		t.Log(rec.Body.String())
	}
}

func TestGetMessagesByChannelID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	for i := 0; i < 5; i++ {
		mustMakeMessage(t)
	}

	q := make(url.Values)
	q.Set("offset", "3")
	q.Set("count", "1")
	req := httptest.NewRequest("GET", "/?"+q.Encode(), nil)

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/messages")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannelID)
	requestWithContext(t, mw(GetMessagesByChannelID), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var responseBody []MessageForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 3)
		}
	}
}

func TestPostMessage(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	post := requestMessage{
		Text: "test message",
	}
	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/messages")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannelID)
	requestWithContext(t, mw(PostMessage), c)

	if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
		message := &MessageForResponse{}
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), message)) {
			assert.Equal(post.Text, message.Content)
		}
	}
}

func TestPutMessageByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	message := mustMakeMessage(t)

	post := requestMessage{
		Text: "test message",
	}
	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(PutMessageByID), c)

	message, err = model.GetMessage(message.ID)
	require.NoError(t, err)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		assert.Equal(post.Text, message.Text)
	}
}

func TestDeleteMessageByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	message := mustMakeMessage(t)

	req := httptest.NewRequest("DELETE", "http://test", nil)

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(DeleteMessageByID), c)

	message, err := model.GetMessage(message.ID)
	require.NoError(t, err)

	if assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String()) {
		assert.True(message.IsDeleted)
	}
}
