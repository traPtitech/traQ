package router

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"
)

func TestGroup_Stars(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	t.Run("TestGetStars", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		mustStarChannel(t, user.ID, channel.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me/stars").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me/stars").
				WithCookie(sessions.CookieName, generateSession(t, user.ID)).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				ContainsOnly(channel.ID.String())
		})
	})

	t.Run("TestPutStars", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
				WithCookie(sessions.CookieName, generateSession(t, user.ID)).
				Expect().
				Status(http.StatusNoContent)

			a, err := model.GetStaredChannels(user.ID)
			require.NoError(err)
			assert.Len(a, 1)
			assert.Contains(a, channel.ID.String())
		})
	})

	t.Run("TestDeleteStars", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		mustStarChannel(t, user.ID, channel.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
				WithCookie(sessions.CookieName, generateSession(t, user.ID)).
				Expect().
				Status(http.StatusNoContent)
			a, err := model.GetStaredChannels(user.ID)
			require.NoError(err)
			assert.Empty(a)
		})
	})
}
