package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"strings"
	"testing"
)

func TestHandlers_PutMyPassword(t *testing.T) {
	t.Parallel()
	path := "/api/v3/users/me/password"
	repo, server := Setup(t, common)
	commonSession := S(t, User(t, repo, random).ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			WithCookie(sessions.CookieName, commonSession).
			WithJSON(echo.Map{"password": 111, "newPassword": false}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password1", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			WithCookie(sessions.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": "a"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password2", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			WithCookie(sessions.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password3", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			WithCookie(sessions.CookieName, commonSession).
			WithJSON(echo.Map{"password": "test", "newPassword": strings.Repeat("a", 33)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		e.PUT(path).
			WithCookie(sessions.CookieName, commonSession).
			WithJSON(echo.Map{"password": "wrong password", "newPassword": strings.Repeat("a", 20)}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		user := User(t, repo, random)

		e := R(t, server)
		new := strings.Repeat("a", 20)
		e.PUT(path).
			WithCookie(sessions.CookieName, S(t, user.ID)).
			WithJSON(echo.Map{"password": "testtesttesttest", "newPassword": new}).
			Expect().
			Status(http.StatusNoContent)

		u, err := repo.GetUser(user.ID)
		require.NoError(t, err)
		assert.NoError(t, model.AuthenticateUser(u, new))
	})
}
