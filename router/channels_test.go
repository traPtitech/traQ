package router

import (
	"bytes"
	"encoding/json"
	"github.com/satori/go.uuid"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGetChannels(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeChannelDetail(t, testUser.GetUID(), "Channel-"+strconv.Itoa(i), "")
	}
	mustMakePrivateChannel(t, "private", []uuid.UUID{testUser.GetUID()})

	rec := request(e, t, mw(GetChannels), cookie, nil)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var res []ChannelForResponse
		assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res))
		assert.Len(res, 6+2)
	}
}

func TestPostChannels(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	postBody := PostChannel{
		Name:   "test",
		Parent: "",
	}

	body, err := json.Marshal(postBody)
	require.NoError(err)
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostChannels), cookie, req)

	channelList, err := model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 3)
		}
	}

	postBody = PostChannel{
		Name:   "test-2",
		Parent: channelList[0].ID.String(),
	}

	body, err = json.Marshal(postBody)
	require.NoError(err)
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PostChannels), cookie, req)

	channelList, err = model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 4)
		}
	}

	recieverID := mustCreateUser(t, "testPostChannels").GetUID()
	postBody = PostChannel{
		Private: true,
		Name:    "testprivate",
		Parent:  "",
		Members: []uuid.UUID{
			testUser.GetUID(),
			recieverID,
		},
	}
	body, err = json.Marshal(postBody)
	require.NoError(err)
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannels), cookie, req)

	channelList, err = model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		assert.Len(channelList, 5)
	}

	channelList, err = model.GetChannelList(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(channelList, 4)
	}
}

func TestGetChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID.String())

	requestWithContext(t, mw(GetChannelByChannelID), c)
	assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String())
}

func TestPatchChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")

	jsonBody := struct {
		Name       string `json:"name"`
		Visibility bool   `json:"visibility"`
	}{
		Name:       "renamed",
		Visibility: true,
	}
	body, err := json.Marshal(jsonBody)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID.String())
	requestWithContext(t, mw(PatchChannelByChannelID), c)

	assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestPutChannelParent(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")

	parentID := mustMakeChannelDetail(t, testUser.GetUID(), "parent", "").ID
	jsonBody := struct {
		Parent string `json:"parent"`
	}{
		Parent: parentID.String(),
	}
	body, err := json.Marshal(jsonBody)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID.String())
	requestWithContext(t, mw(PutChannelParent), c)

	assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestDeleteChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "")

	c, _ := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID.String())
	requestWithContext(t, mw(DeleteChannelByChannelID), c)

	ch, err := model.GetChannelWithUserID(testUser.GetUID(), ch.ID)
	require.Error(err)

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		assert.Len(channelList, 2)
	}
}
