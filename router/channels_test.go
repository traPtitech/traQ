package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGetChannels(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeChannel(t, testUser.ID, "Channel-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannels), cookie, nil)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var responseBody []ChannelForResponse
		assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody))
	}
}

func TestPostChannels(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	postBody := PostChannel{
		ChannelType: "public",
		Name:        "test",
		Parent:      "",
	}

	body, err := json.Marshal(postBody)
	require.NoError(err)
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostChannels), cookie, req)

	channelList, err := model.GetChannels(testUser.ID)
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 1)
		}
	}

	postBody = PostChannel{
		ChannelType: "public",
		Name:        "test-2",
		Parent:      channelList[0].ID,
	}

	body, err = json.Marshal(postBody)
	require.NoError(err)
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PostChannels), cookie, req)

	channelList, err = model.GetChannels(testUser.ID)
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 2)
			assert.False(channelList[0].ID != channelList[1].ParentID && channelList[1].ID != channelList[0].ParentID)
		}
	}

	postBody = PostChannel{
		ChannelType: "private",
		Name:        "testprivate",
		Parent:      "",
		Member: []string{
			testUser.ID,
			mustCreateUser(t, "testPostChannels").ID,
		},
	}
	body, err = json.Marshal(postBody)
	require.NoError(err)
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannels), cookie, req)

	channelList, err = model.GetChannels(testUser.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 3)
	}

	channelList, err = model.GetChannels(model.CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 2)
	}
}

func TestGetChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)

	requestWithContext(t, mw(GetChannelsByChannelID), c)
	assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String())
}

func TestPutChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)

	req := httptest.NewRequest("PUT", "http://test", strings.NewReader(`{"name": "renamed"}`))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutChannelsByChannelID), c)

	require.EqualValues(http.StatusOK, rec.Code, rec.Body.String())

	channel, err := model.GetChannelByID(testUser.ID, channel.ID)
	if assert.NoError(err) {
		assert.Equal("renamed", channel.Name)
		assert.Equal(testUser.ID, channel.UpdaterID)
	}
}

func TestDeleteChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)

	req := httptest.NewRequest("DELETE", "http://test", strings.NewReader(`{"confirm": true}`))
	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelID), c)

	channel, err := model.GetChannelByID(testUser.ID, channel.ID)
	require.Error(err)

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannels(testUser.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 0)
	}
}
