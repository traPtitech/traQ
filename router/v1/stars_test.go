package v1

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_GetStars(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common3)

	channel := env.mustMakeChannel(t, rand)
	env.mustStarChannel(t, testUser.GetID(), channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/me/stars").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/me/stars").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(channel.ID.String())
	})
}

func TestHandlers_PutStars(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common3)

	channel := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		a, err := env.Repository.GetStaredChannels(testUser.GetID())
		require.NoError(t, err)
		assert.Len(t, a, 1)
		assert.Contains(t, a, channel.ID)
	})
}

func TestHandlers_DeleteStars(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common3)

	channel := env.mustMakeChannel(t, rand)
	env.mustStarChannel(t, testUser.GetID(), channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/me/stars/{channelID}", channel.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
		a, err := env.Repository.GetStaredChannels(testUser.GetID())
		require.NoError(t, err)
		assert.Empty(t, a)
	})
}
