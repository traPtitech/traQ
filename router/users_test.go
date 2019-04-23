package router

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"gopkg.in/guregu/null.v3"
	"strings"
	"testing"

	"net/http"
)

func TestHandlers_GetUsers(t *testing.T) {
	t.Parallel()
	_, server, _, _, session, _ := setup(t, s2)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(2)
	})
}

func TestHandlers_GetMe(t *testing.T) {
	t.Parallel()
	_, server, _, _, session, _, testUser, _ := setupWithUsers(t, common4)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("userId").
			String().
			Equal(testUser.ID.String())
	})
}

func TestHandlers_GetUserByID(t *testing.T) {
	t.Parallel()
	_, server, _, _, session, _, testUser, _ := setupWithUsers(t, common4)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}", testUser.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}", testUser.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("userId").
			String().
			Equal(testUser.ID.String())
	})
}

func TestHandlers_PatchMe(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common4)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/me").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		newDisp := "renamed"
		newTwitter := "test"
		e.PATCH("/api/1.0/users/me").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"displayName": newDisp, "twitterId": newTwitter}).
			Expect().
			Status(http.StatusNoContent)

		u, err := repo.GetUser(user.ID)
		require.NoError(t, err)
		assert.Equal(t, newDisp, u.DisplayName)
		assert.Equal(t, newTwitter, u.TwitterID)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		user := mustMakeUser(t, repo, random)
		require.NoError(t, repo.UpdateUser(user.ID, repository.UpdateUserArgs{DisplayName: null.StringFrom("test")}))

		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/me").
			WithCookie(sessions.CookieName, generateSession(t, user.ID)).
			WithJSON(map[string]string{"displayName": ""}).
			Expect().
			Status(http.StatusNoContent)

		u, err := repo.GetUser(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "", u.DisplayName)
	})
}

func TestHandlers_PutPassword(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common4)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]interface{}{"password": 111, "newPassword": false}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"password": "test", "newPassword": "a"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"password": "test", "newPassword": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password3", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"password": "test", "newPassword": strings.Repeat("a", 33)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"password": "wrong password", "newPassword": strings.Repeat("a", 20)}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		user := mustMakeUser(t, repo, random)

		e := makeExp(t, server)
		new := strings.Repeat("a", 20)
		e.PUT("/api/1.0/users/me/password").
			WithCookie(sessions.CookieName, generateSession(t, user.ID)).
			WithJSON(map[string]string{"password": "test", "newPassword": new}).
			Expect().
			Status(http.StatusNoContent)

		u, err := repo.GetUser(user.ID)
		require.NoError(t, err)
		assert.NoError(t, model.AuthenticateUser(u, new))
	})
}

func TestHandlers_PostLogin(t *testing.T) {
	t.Parallel()
	_, server, _, _, _, _, user, _ := setupWithUsers(t, common4)

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/login").
			WithJSON(map[string]string{"name": user.Name, "pass": "test"}).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/login").
			WithJSON(map[string]string{"name": user.Name, "pass": "wrong_password"}).
			Expect().
			Status(http.StatusUnauthorized)
	})
}
