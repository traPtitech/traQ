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
	"github.com/traPtitech/traQ/model"
)

func TestGetChannelsHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	for i := 0; i < 5; i++ {
		makeChannel(testUserID, "Channel-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannels), cookie, nil)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	var responseBody []ChannelForResponse
	err := json.Unmarshal(rec.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatal("Failed to json parse ", err)
	}
}

func TestPostChannelsHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	postBody := PostChannel{
		ChannelType: "public",
		Name:        "test",
		Parent:      "",
	}

	body, err := json.Marshal(postBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostChannels), cookie, req)

	channelList, err := model.GetChannels(testUserID)

	if err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusCreated {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	if len(channelList) != 1 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 1, len(channelList))
	}

	postBody = PostChannel{
		ChannelType: "private",
		Name:        "test",
		Parent:      "",
		Member: []string{
			testUserID,
			model.CreateUUID(),
		},
	}
	body, err = json.Marshal(postBody)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannels), cookie, req)
	channelList, err = model.GetChannels(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(channelList) != 2 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 2, len(channelList))
	}

	req = httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	request(e, t, mw(PostChannels), cookie, req)
	channelList, err = model.GetChannels(model.CreateUUID())
	if err != nil {
		t.Fatal(err)
	}

	if len(channelList) != 1 {
		t.Fatalf("Channel List wrong: want %d, actual %d\n", 1, len(channelList))
	}
}

func TestGetChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(testUserID, "test", true)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)

	requestWithContext(t, mw(GetChannelsByChannelID), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	t.Log(rec.Body.String())
}

func TestPutChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(model.CreateUUID(), "test", true)

	req := httptest.NewRequest("PUT", "http://test", strings.NewReader(`{"name": "renamed"}`))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(PutChannelsByChannelID), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}

	channel, err := model.GetChannelByID(testUserID, channel.ID)
	if err != nil {
		t.Fatal(err)
	}

	if channel.Name != "renamed" {
		t.Fatalf("Channel name wrong: want %s, actual %s", "renamed", channel.Name)
	}

	if channel.UpdaterID != testUserID {
		t.Fatalf("Channel UpdaterId wrong: want %s, actual %s", testUserID, channel.UpdaterID)
	}

}

func TestDeleteChannelsByChannelIDHandler(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	channel, _ := makeChannel(model.CreateUUID(), "test", true)

	req := httptest.NewRequest("DELETE", "http://test", strings.NewReader(`{"confirm": true}`))
	c, _ := getContext(e, t, cookie, req)
	c.SetPath("/:channelId")
	c.SetParamNames("channelId")
	c.SetParamValues(channel.ID)
	requestWithContext(t, mw(DeleteChannelsByChannelID), c)

	channel, err := model.GetChannelByID(testUserID, channel.ID)

	if err == nil {
		t.Fatal("The channel that was supposed to be deleted is displayed to the user")
	}

	// ""で削除されていても取得できるようにするそれでちゃんと削除されているか確認する

	channelList, err := model.GetChannels(testUserID)
	if len(channelList) != 0 {
		t.Fatal("Channel not deleted")
	}
}
