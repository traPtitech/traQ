package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
)

func TestGetChannels(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeChannel(t, testUser.ID, "Channel-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannels), cookie, nil)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	var responseBody []ChannelForResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
}

func TestPostChannels(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	require := require.New(t)
	assert := assert.New(t)

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
		Name:        "test",
		Parent:      "",
		Member: []string{
			testUser.ID,
			model.CreateUUID(),
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

	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannels), cookie, req)
	channelList, err = model.GetChannels(model.CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 2)
	}
}

func TestGetChannelsByChannelID(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel := mustMakeChannel(t, testUser.ID, "test", true)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)

	requestWithContext(t, mw(GetChannelsByChannelID), c)
	if assert.EqualValues(t, http.StatusOK, rec.Code, rec.Body.String()) {
		t.Log(rec.Body.String())
	}
}

func TestPutChannelsByChannelID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)
	channel := mustMakeChannel(t, model.CreateUUID(), "test", true)

	req := httptest.NewRequest("PUT", "http://test", strings.NewReader(`{"name": "renamed"}`))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutChannelsByChannelID), c)

	require.EqualValues(t, http.StatusOK, rec.Code, rec.Body.String())

	channel, err := model.GetChannelByID(testUser.ID, channel.ID)
	if assert.NoError(err) {
		assert.Equal("renamed", channel.Name)
		assert.Equal(testUser.ID, channel.UpdaterID)
	}
}

func TestDeleteChannelsByChannelID(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel := mustMakeChannel(t, model.CreateUUID(), "test", true)

	req := httptest.NewRequest("DELETE", "http://test", strings.NewReader(`{"confirm": true}`))
	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelID), c)

	channel, err := model.GetChannelByID(testUser.ID, channel.ID)
	require.Error(t, err)

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannels(testUser.ID)
	if assert.NoError(t, err) {
		assert.Len(t, channelList, 0)
	}
}
