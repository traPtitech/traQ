package router

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/traPtitech/traQ/model"
)

//TODO TestPutNotificationStatus
//TODO TestPostDeviceToken
//TODO TestDeleteDeviceToken
//TODO TestGetNotificationStream

func TestGetNotificationStatus(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channelID := mustMakeChannel(t, testUser.ID, "subscribing", true).ID
	userID := mustCreateUser(t, "poyo").ID

	usc := model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}
	require.NoError(usc.Create())
	usc = model.UserSubscribeChannel{UserID: testUser.ID, ChannelID: channelID}
	require.NoError(usc.Create())

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:channelID")
	c.SetParamNames("channelID")
	c.SetParamValues(channelID)

	requestWithContext(t, mw(GetNotificationStatus), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var res []string
		require.NoError(json.Unmarshal(rec.Body.Bytes(), &res))
		assert.Len(res, 2)
	}
}

func TestGetNotificationChannels(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	channelID := mustMakeChannel(t, testUser.ID, "subscribing", true).ID
	usc := model.UserSubscribeChannel{UserID: testUser.ID, ChannelID: channelID}
	require.NoError(usc.Create())

	channelID = mustMakeChannel(t, testUser.ID, "subscribing2", true).ID
	usc = model.UserSubscribeChannel{UserID: testUser.ID, ChannelID: channelID}
	require.NoError(usc.Create())

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:userID")
	c.SetParamNames("userID")
	c.SetParamValues(testUser.ID)

	requestWithContext(t, mw(GetNotificationChannels), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var res []ChannelForResponse
		require.NoError(json.Unmarshal(rec.Body.Bytes(), &res))
		assert.Len(res, 2)
	}
}
