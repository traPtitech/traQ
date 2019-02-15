package router

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"testing"
)

func TestHandlers_PostPin(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/pins").
			WithJSON(map[string]string{"messageId": message.ID.String()}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
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

		p, err := repo.GetPinsByChannelID(channel.ID)
		require.NoError(t, err)
		assert.Len(t, p, 1)
	})
}

func TestHandlers_GetPin(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	pin := mustMakePin(t, repo, message.ID, testUser.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/pins/{pinID}", pin.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
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
}

func TestHandlers_DeletePin(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	pin := mustMakePin(t, repo, message.ID, testUser.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/pins/{pinID}", pin.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/pins/{pinID}", pin.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetPin(pin)
		assert.Equal(t, repository.ErrNotFound, err)
	})
}

func TestHandlers_GetChannelPin(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	mustMakePin(t, repo, message.ID, testUser.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/pins", channel.ID.String()).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/pins", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(1)
	})
}
