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
		mustMakeChannelDetail(t, testUser.GetUID(), "Channel-"+strconv.Itoa(i), "", true)
	}
	private := mustMakeChannelDetail(t, testUser.GetUID(), "private", "", false)

	rec := request(e, t, mw(GetChannels), cookie, nil)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var res []ChannelForResponse
		assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res))
		for _, v := range res {
			if v.ChannelID == private.ID {
				assert.Equal(privateParentChannelID, v.Parent)
				assert.Len(v.Member, 2)
			}
		}
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

	channelList, err := model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 3)
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

	channelList, err = model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 4)
		}
	}

	recieverID := mustCreateUser(t, "testPostChannels").ID
	postBody = PostChannel{
		ChannelType: "private",
		Name:        "testprivate",
		Parent:      privateParentChannelID,
		Member: []string{
			testUser.ID,
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

	// 異常系: 同じメンバーのプライベートチャンネルは作成できない
	postBody = PostChannel{
		ChannelType: "private",
		Name:        "testprivate-error",
		Parent:      privateParentChannelID,
		Member: []string{
			testUser.ID,
			recieverID,
		},
	}
	body, err = json.Marshal(postBody)
	require.NoError(err)

	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PostChannels), cookie, req)

	assert.Equal(http.StatusBadRequest, rec.Code)
}

func TestGetChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channel.ID)

	requestWithContext(t, mw(GetChannelsByChannelID), c)
	assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String())
}

func TestPatchChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)

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
	c.SetParamValues(ch.ID)
	requestWithContext(t, mw(PatchChannelsByChannelID), c)

	assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestPutChannelParent(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)

	parentID := mustMakeChannelDetail(t, testUser.GetUID(), "parent", "", true).ID
	jsonBody := struct {
		Parent string `json:"parent"`
	}{
		Parent: parentID,
	}
	body, err := json.Marshal(jsonBody)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID)
	requestWithContext(t, mw(PutChannelParent), c)

	assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestDeleteChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	ch := mustMakeChannelDetail(t, testUser.GetUID(), "test", "", true)

	c, _ := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelID), c)

	ch, err := model.GetChannelWithUserID(testUser.GetUID(), ch.GetCID())
	require.Error(err)

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannelList(testUser.GetUID())
	if assert.NoError(err) {
		assert.Len(channelList, 2)
	}
}
