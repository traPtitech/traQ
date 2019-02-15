package router

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/sessions"
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
			Status(http.StatusForbidden)
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
	_, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me").
			Expect().
			Status(http.StatusForbidden)
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
	_, server, _, _, session, _, testUser, _ := setupWithUsers(t, common)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}", testUser.ID.String()).
			Expect().
			Status(http.StatusForbidden)
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
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/me").
			Expect().
			Status(http.StatusForbidden)
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
}

func TestHandlers_PostLogin(t *testing.T) {
	t.Parallel()
	_, server, _, _, _, _, user, _ := setupWithUsers(t, common)

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/login").
			WithJSON(map[string]string{"name": user.Name, "pass": "test"}).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/login").
			WithJSON(map[string]string{"name": user.Name, "pass": "wrong_password"}).
			Expect().
			Status(http.StatusForbidden)
	})
}
