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
	beforeTest(t)
	topicText := "Topic test"
	e, cookie, mw := beforeTest(t)

	channel, err := makeChannel(testUserID, "putTopicTest", true)
	if err != nil {
		t.Fatal(err)
	}
	channel.Topic = topicText

	if err := channel.Update(); err != nil {
		t.Fatal(err)
	}

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(GetTopic), c)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status code is not 200, actual %d", rec.Code)
	}

	responseBody := TopicForResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Error while json unmarshal: %v", err)
	}

	if responseBody.ChannelID != channel.ID {
		t.Fatalf("ChannelID is wrong, want %s, actual %s", channel.ID, responseBody.ChannelID)
	}

	if responseBody.Name != channel.Name {
		t.Fatalf("Channel name is wrong, want %s, actual %s", channel.Name, responseBody.Name)
	}

	if responseBody.Text != topicText {
		t.Fatalf("Topic text is wrong, want %s, actual %s", topicText, responseBody.Text)
	}
}

func TestPutTopic(t *testing.T) {
	topicText := "Topic test"
	e, cookie, mw := beforeTest(t)

	channel, err := makeChannel(model.CreateUUID(), "putTopicTest", true)
	if err != nil {
		t.Fatal(err)
	}

	if channel.Topic != "" {
		t.Fatal("channel topic is not empty")
	}

	type putTopic struct {
		Text string `json:"text"`
	}

	requestBody := &putTopic{
		Text: topicText,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutTopic), c)

	if rec.Code != http.StatusOK {
		t.Fatalf("Status code is not 200, actual %d", rec.Code)
	}
	check := &model.Channel{
		ID: channel.ID,
	}
	check.Exists(testUserID)
	t.Log(check)

	responseBody := &TopicForResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), responseBody); err != nil {
		t.Fatalf("Error while json unmarshal: %v", err)
	}

	if responseBody.ChannelID != channel.ID {
		t.Fatalf("ChannelID is wrong, want %s, actual %s", channel.ID, responseBody.ChannelID)
	}

	if responseBody.Name != channel.Name {
		t.Fatalf("Channel name is wrong, want %s, actual %s", channel.Name, responseBody.Name)
	}

	if responseBody.Text != topicText {
		t.Fatalf("Topic text is wrong, want %s, actual %s", topicText, responseBody.Text)
	}

	if check.Topic != topicText {
		t.Fatalf("Topic text is wrong, want %s, actual %s", topicText, channel.Topic)
	}

	if check.UpdaterID != testUserID {
		t.Fatalf("UpdaterID is wrong, want %s, actual %s", testUserID, channel.UpdaterID)
	}
}
