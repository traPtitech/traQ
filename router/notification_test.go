package router

import (
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

func TestGroup_Notification(t *testing.T) {
	assert, require, session, _ := beforeTest(t)

	t.Run("TestPutNotificationStatus", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/notifications", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/channels/{channelID}/notifications", channel.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string][]string{"on": {user.ID}}).
				Expect().
				Status(http.StatusNoContent)

			users, err := model.GetSubscribingUser(channel.ID)
			require.NoError(err)
			assert.EqualValues(users, []uuid.UUID{user.GetUID()})
		})
	})

	t.Run("TestGetNotificationStatus", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))

		require.NoError(model.SubscribeChannel(user.GetUID(), channel.ID))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/notifications", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/notifications", channel.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(1)
		})
	})

	t.Run("TestGetNotificationChannels", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(model.SubscribeChannel(user.GetUID(), mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "").ID))
		require.NoError(model.SubscribeChannel(user.GetUID(), mustMakeChannelDetail(t, testUser.GetUID(), utils.RandAlphabetAndNumberString(20), "").ID))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}/notification", user.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}/notification", user.ID).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(2)
		})
	})
}
