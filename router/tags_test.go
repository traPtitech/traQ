package router

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"
)

func TestGroup_Tags(t *testing.T) {
	assert, require, session, _ := beforeTest(t)

	t.Run("TestPostUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/users/{userID}/tags", user.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			tag := utils.RandAlphabetAndNumberString(20)
			e.POST("/api/1.0/users/{userID}/tags", user.ID).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"tag": tag}).
				Expect().
				Status(http.StatusCreated)

			a, err := model.GetUserIDsByTag(tag)
			require.NoError(err)
			assert.Len(a, 1)
			assert.Contains(a, user.GetUID())
		})
	})

	t.Run("TestGetUserTags", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		for i := 0; i < 5; i++ {
			mustMakeTag(t, user.GetUID(), utils.RandAlphabetAndNumberString(20))
		}

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}/tags", user.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/{userID}/tags", user.ID).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(5)
		})
	})

	t.Run("TestPatchUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, user.GetUID(), utils.RandAlphabetAndNumberString(20))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID, tag.String()).
				WithJSON(map[string]bool{"isLocked": true}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID, tag.String()).
				WithCookie(sessions.CookieName, generateSession(t, user.GetUID())).
				WithJSON(map[string]bool{"isLocked": true}).
				Expect().
				Status(http.StatusNoContent)

			ut, err := model.GetUserTag(user.GetUID(), tag)
			require.NoError(err)
			assert.True(ut.IsLocked)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID, tag.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]bool{"isLocked": true}).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestDeleteUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, user.GetUID(), utils.RandAlphabetAndNumberString(20))

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.ID, tag.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.ID, tag.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNoContent)

			_, err := model.GetUserTag(user.GetUID(), tag)
			require.Equal(model.ErrNotFound, err)
		})
	})
}
