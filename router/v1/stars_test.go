package v1

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/router/sessions"
	"net/http"
	"testing"
)

func TestHandlers_GetStars(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common3)

	channel := mustMakeChannel(t, repo, random)
	mustStarChannel(t, repo, testUser.GetID(), channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/stars").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/stars").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(channel.ID.String())
	})
}

func TestHandlers_PutStars(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common3)

	channel := mustMakeChannel(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		a, err := repo.GetStaredChannels(testUser.GetID())
		require.NoError(t, err)
		assert.Len(t, a, 1)
		assert.Contains(t, a, channel.ID)
	})
}

func TestHandlers_DeleteStars(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common3)

	channel := mustMakeChannel(t, repo, random)
	mustStarChannel(t, repo, testUser.GetID(), channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)
		a, err := repo.GetStaredChannels(testUser.GetID())
		require.NoError(t, err)
		assert.Empty(t, a)
	})
}
