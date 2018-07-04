package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

func TestGetMessageByID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)
	message := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)

	requestWithContext(t, mw(GetMessageByID), c)

	assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String())

	// 異常系: 自分から見えないメッセージは取得できない
	postmanID := mustCreateUser(t, "p1").GetUID()
	privateID := mustMakePrivateChannel(t, postmanID, mustCreateUser(t, "p2").GetUID(), "private").GetCID()
	message = mustMakeMessage(t, postmanID, privateID)

	c, rec = getContext(e, t, cookie, nil)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)

	err := mw(GetMessageByID)(c)

	if assert.Error(err) {
		assert.Equal(http.StatusNotFound, err.(*echo.HTTPError).Code)
	}
}

func TestGetMessagesByChannelID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)
	for i := 0; i < 5; i++ {
		mustMakeMessage(t, testUser.GetUID(), channel.GetCID())
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

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)

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

	user1ID := mustCreateUser(t, "private-1").GetUID()
	user2ID := mustCreateUser(t, "private-2").GetUID()
	privateID := mustMakePrivateChannel(t, user1ID, user2ID, "poyopoyo").ID

	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	c, rec = getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/messages")
	c.SetParamNames("channelID")
	c.SetParamValues(privateID)

	err = mw(PostMessage)(c)

	if assert.Error(err) {
		assert.Equal(http.StatusNotFound, err.(*echo.HTTPError).Code)
	}
}

func TestPutMessageByID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)
	message := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	post := struct{ Text string }{Text: "test message"}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))

	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(PutMessageByID), c)

	message, err = model.GetMessageByID(message.GetID())
	require.NoError(err)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		assert.Equal(post.Text, message.Text)
	}

	// 異常系：他人のメッセージは編集できない
	creatorID := mustCreateUser(t, "creator").GetUID()
	message = mustMakeMessage(t, creatorID, channel.GetCID())

	req = httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))

	c, rec = getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)

	err = mw(PutMessageByID)(c)

	if assert.Error(err) {
		assert.Equal(http.StatusForbidden, err.(*echo.HTTPError).Code)
	}

}

func TestDeleteMessageByID(t *testing.T) {
	e, cookie, mw, _, require := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)
	message := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	req := httptest.NewRequest("DELETE", "http://test", nil)

	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/messages/:messageID")
	c.SetParamNames("messageID")
	c.SetParamValues(message.ID)
	requestWithContext(t, mw(DeleteMessageByID), c)

	message, err := model.GetMessageByID(message.GetID())
	require.Error(err)
}

func TestPostMessageReport(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)
	message := mustMakeMessage(t, testUser.GetUID(), channel.GetCID())

	{
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		c, _ := getContext(e, t, cookie, req)
		c.SetPath("/messages/:messageID/report")
		c.SetParamNames("messageID")
		c.SetParamValues(message.ID)

		err := mw(PostMessageReport)(c)

		if assert.Error(err) {
			assert.Equal(http.StatusBadRequest, err.(*echo.HTTPError).Code)
		}
	}

	{
		post := struct{ Reason string }{Reason: "ああああ"}
		body, err := json.Marshal(post)
		require.NoError(err)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

		c, _ := getContext(e, t, cookie, req)
		c.SetPath("/messages/:messageID/report")
		c.SetParamNames("messageID")
		c.SetParamValues(message.ID)
		requestWithContext(t, mw(PostMessageReport), c)

		_, err = model.GetMessageByID(message.GetID())
		assert.NoError(err)
	}
}
