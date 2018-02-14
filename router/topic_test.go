package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
)

func TestGetTopic(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	topicText := "Topic test"

	channel := mustMakeChannel(t, testUser.ID, "putTopicTest", true)
	channel.Topic = topicText
	require.NoError(t, channel.Update())

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(GetTopic), c)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		responseBody := TopicForResponse{}
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Equal(channel.ID, responseBody.ChannelID)
			assert.Equal(channel.Name, responseBody.Name)
			assert.Equal(topicText, responseBody.Text)
		}
	}
}

func TestPutTopic(t *testing.T) {
	topicText := "Topic test"
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	channel := mustMakeChannel(t, model.CreateUUID(), "putTopicTest", true)
	require.Empty(t, channel.Topic)

	type putTopic struct {
		Text string `json:"text"`
	}

	requestBody := &putTopic{
		Text: topicText,
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutTopic), c)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		check := &model.Channel{
			ID: channel.ID,
		}
		check.Exists(testUser.ID)
		t.Log(check)

		responseBody := &TopicForResponse{}
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), responseBody)) {
			assert.Equal(channel.ID, responseBody.ChannelID)
			assert.Equal(channel.Name, responseBody.Name)
			assert.Equal(topicText, responseBody.Text)
			assert.Equal(topicText, check.Topic)
			assert.Equal(testUser.ID, check.UpdaterID)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), responseBody); err != nil {
			t.Fatalf("Error while json unmarshal: %v", err)
		}
	}
}
