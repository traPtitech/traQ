package v3

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_PutMyNotifyCitation(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/settings/notify-citation"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithJSON(&PutMyNotifyCitationRequest{NotifyCitation: true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PutMyNotifyCitationRequest{NotifyCitation: true}).
			Expect().
			Status(http.StatusNoContent)

		nc, err := env.Repository.GetNotifyCitation(user.GetID())
		require.NoError(t, err)
		assert.True(t, nc)
	})
}

func TestHandlers_GetMySettings(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/settings"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
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
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().IsEqual(user.GetID().String())
		obj.Value("notifyCitation").Boolean().IsFalse()
	})
}

func TestHandlers_GetMyNotifyCitation(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/settings/notify-citation"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
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
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("notifyCitation").Boolean().IsFalse()
	})
}
