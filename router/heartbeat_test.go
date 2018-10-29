package router

import (
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"testing"
	"time"

	"github.com/traPtitech/traQ/model"
	"net/http"
)

func TestGroup_Heartbeat(t *testing.T) {
	_, _, session, _ := beforeTest(t)

	t.Run("TestPostHeartbeat", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/heartbeat").
				WithJSON(map[string]string{"channelId": channel.ID.String(), "status": "editing"}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			obj := e.POST("/api/1.0/heartbeat").
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"channelId": channel.ID.String(), "status": "editing"}).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			obj.Value("channelId").String().Equal(channel.ID.String())
			obj.Value("userStatuses").Array().Length().Equal(1)
			obj.Value("userStatuses").Array().First().Object().Value("userId").Equal(testUser.ID)
			obj.Value("userStatuses").Array().First().Object().Value("status").Equal("editing")
		})
	})

	t.Run("TestGetHeartbeat", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		model.HeartbeatStatuses[channel.ID] = &model.HeartbeatStatus{
			ChannelID: channel.ID,
			UserStatuses: []*model.UserStatus{
				{
					UserID:   testUser.GetUID(),
					Status:   "editing",
					LastTime: time.Now(),
				},
			},
		}

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/heartbeat").
				WithQuery("channelId", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			obj := e.POST("/api/1.0/heartbeat").
				WithCookie(sessions.CookieName, session).
				WithQuery("channelId", channel.ID.String()).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			obj.Value("channelId").String().Equal(channel.ID.String())
			obj.Value("userStatuses").Array().Length().Equal(1)
			obj.Value("userStatuses").Array().First().Object().Value("userId").Equal(testUser.ID)
			obj.Value("userStatuses").Array().First().Object().Value("status").Equal("editing")
		})
	})
}
