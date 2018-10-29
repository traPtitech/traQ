package router

import (
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"testing"

	"github.com/traPtitech/traQ/model"

	"net/http"
)

func TestGroup_Users(t *testing.T) {
	assert, require, session, _ := beforeTest(t)

	t.Run("TestGetUsers", func(t *testing.T) {
		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users").
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(2)
		})
	})

	// ここから並列テスト

	t.Run("TestGetMe", func(t *testing.T) {
		t.Parallel()

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me").
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().
				Value("userId").
				String().
				Equal(testUser.ID)
		})
	})

	t.Run("TestGetUserByID", func(t *testing.T) {
		t.Parallel()

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}", testUser.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}", testUser.ID).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().
				Value("userId").
				String().
				Equal(testUser.ID)
		})
	})

	t.Run("TestPatchMe", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/users/me").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			newDisp := "renamed"
			newTwitter := "test"
			e.PATCH("/api/1.0/users/me").
				WithCookie(sessions.CookieName, generateSession(t, user.GetUID())).
				WithJSON(map[string]string{"displayName": newDisp, "twitterId": newTwitter}).
				Expect().
				Status(http.StatusNoContent)

			u, err := model.GetUser(user.GetUID())
			require.NoError(err)
			assert.Equal(newDisp, u.DisplayName)
			assert.Equal(newTwitter, u.TwitterID)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/users/me").
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"displayName": "a", "twitterId": "a"}).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestPostLogin", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/login").
				WithJSON(map[string]string{"name": user.Name, "pass": "test"}).
				Expect().
				Status(http.StatusNoContent)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/login").
				WithJSON(map[string]string{"name": user.Name, "pass": "wrong_password"}).
				Expect().
				Status(http.StatusForbidden)
		})
	})
}
