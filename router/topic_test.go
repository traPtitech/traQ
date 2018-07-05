package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGetTopic(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	topicText := "Topic test"

	ch := mustMakeChannelDetail(t, testUser.GetUID(), "putTopicTest", "", true)
	require.NoError(model.UpdateChannelTopic(ch.GetCID(), topicText, testUser.GetUID()))

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID)
	requestWithContext(t, mw(GetTopic), c)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		responseBody := struct {
			Text string `json:"text"`
		}{}
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Equal(topicText, responseBody.Text)
		}
	}
}

func TestPutTopic(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	topicText := "Topic test"

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "putTopicTest", "", true)
	require.Empty(channel.Topic)

	type putTopic struct {
		Text string `json:"text"`
	}

	requestBody := &putTopic{
		Text: topicText,
	}
	body, err := json.Marshal(requestBody)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutTopic), c)

	if assert.EqualValues(http.StatusNoContent, rec.Code) {
		ch, err := model.GetChannel(channel.GetCID())
		require.NoError(err)
		assert.Equal(topicText, ch.Topic)
	}
}
