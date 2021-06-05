package v3

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_GetMyStars(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/stars"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch1 := env.CreateChannel(t, rand)
	ch2 := env.CreateChannel(t, rand)
	env.CreateChannel(t, rand)

	env.AddStar(t, user.GetID(), ch1.ID)
	env.AddStar(t, user.GetID(), ch2.ID)

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(ch1.ID, ch2.ID)
	})
}

func TestHandlers_PostStar(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/stars"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostStarRequest{ChannelID: ch.ID}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostStarRequest{ChannelID: dm.ID}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostStarRequest{ChannelID: ch.ID}).
			Expect().
			Status(http.StatusNoContent)

		stars, err := env.Repository.GetStaredChannels(user.GetID())
		require.NoError(t, err)
		assert.ElementsMatch(t, stars, []uuid.UUID{ch.ID})
	})
}

func TestHandlers_RemoveMyStar(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/stars/{channelId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch1 := env.CreateChannel(t, rand)
	ch2 := env.CreateChannel(t, rand)

	env.AddStar(t, user.GetID(), ch1.ID)

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ch1.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (already removed)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ch2.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ch1.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		stars, err := env.Repository.GetStaredChannels(user.GetID())
		require.NoError(t, err)
		assert.Len(t, stars, 0)
	})
}
