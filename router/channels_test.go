package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

func TestGetChannels(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeChannel(t, testUser.ID, "Channel-"+strconv.Itoa(i), true)
	}
	private := mustMakeChannel(t, testUser.ID, "private", false)

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

	channelList, err := model.GetChannelList(testUser.ID)
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

	channelList, err = model.GetChannelList(testUser.ID)
	if assert.NoError(err) {
		if assert.EqualValues(http.StatusCreated, rec.Code, rec.Body.String()) {
			assert.Len(channelList, 2)
			assert.False(channelList[0].ID != channelList[1].ParentID && channelList[1].ID != channelList[0].ParentID)
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

	channelList, err = model.GetChannelList(testUser.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 3)
	}

	channelList, err = model.GetChannelList(model.CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 2)
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
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if cookie != nil {
		req.Header.Add("Cookie", fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	rec = httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = mw(PostChannels)(c)

	if assert.Error(err) {
		assert.Equal(http.StatusBadRequest, err.(*echo.HTTPError).Code)
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

func TestPatchChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannel(t, testUser.ID, "test", true)

	parentID := mustMakeChannel(t, testUser.ID, "parent", true).ID
	jsonBody := struct {
		Name       string `json:"name"`
		Parent     string `json:"parent"`
		Visibility bool   `json:"visibility"`
	}{
		Name:       "renamed",
		Parent:     parentID,
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

func TestDeleteChannelsByChannelID(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	ch := mustMakeChannel(t, testUser.ID, "test", true)

	c, _ := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(ch.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelID), c)

	ch, err := model.GetChannelByID(testUser.ID, ch.ID)
	require.Error(err)

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannelList(testUser.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 0)
	}
}
