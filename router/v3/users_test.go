package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_PutMyPassword(t *testing.T) {
	t.Parallel()
	path := "/api/v3/users/me/password"
	env := Setup(t, common1)
	commonSession := env.S(t, env.CreateUser(t, rand).GetID())

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(echo.Map{"password": 111, "newPassword": false}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password1", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": "a"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password2", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password3", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": strings.Repeat("a", 33)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(echo.Map{"password": "wrong password", "newPassword": strings.Repeat("a", 20)}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		user := env.CreateUser(t, rand)

		e := env.R(t)
		newPass := strings.Repeat("a", 20)
		e.PUT(path).
			WithCookie(session.CookieName, env.S(t, user.GetID())).
			WithJSON(echo.Map{"password": "testtesttesttest", "newPassword": newPass}).
			Expect().
			Status(http.StatusNoContent)

		u, err := env.Repository.GetUser(user.GetID(), false)
		require.NoError(t, err)
		assert.NoError(t, u.Authenticate(newPass))
	})
}
