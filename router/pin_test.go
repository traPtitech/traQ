package router

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"
)

func TestGroup_Pin(t *testing.T) {
	assert, require, session, _ := beforeTest(t)

	t.Run("TestPostPin", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/pins").
				WithJSON(map[string]string{"messageId": message.ID.String()}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/pins").
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"messageId": message.ID.String()}).
				Expect().
				Status(http.StatusCreated).
				JSON().
				Object().
				Value("id").
				String().
				NotEmpty()

			p, err := model.GetPinsByChannelID(channel.ID)
			require.NoError(err)
			assert.Len(p, 1)
		})
	})

	t.Run("TestGetPin", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		pin := mustMakePin(t, testUser.ID, message.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/pins/{pinID}", pin.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/pins/{pinID}", pin.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().
				Value("pinId").
				String().
				Equal(pin.String())
		})
	})

	t.Run("TestDeletePin", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		pin := mustMakePin(t, testUser.ID, message.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/pins/{pinID}", pin.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/pins/{pinID}", pin.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNoContent)

			_, err := model.GetPin(pin)
			assert.Equal(model.ErrNotFound, err)
		})
	})

	t.Run("TestGetChannelPin", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		mustMakePin(t, testUser.ID, message.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/pins", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/pins", channel.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(1)
		})
	})
}
