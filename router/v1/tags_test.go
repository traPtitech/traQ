package v1

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestHandlers_PostUserTag(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, user, _ := setupWithUsers(t, common3)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/users/{userID}/tags", user.GetID().String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		tag := random2.AlphaNumeric(20)
		e.POST("/api/1.0/users/{userID}/tags", user.GetID().String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"tag": tag}).
			Expect().
			Status(http.StatusCreated)

		a, err := env.Repository.GetUserTagsByUserID(user.GetID())
		require.NoError(t, err)
		assert.Len(t, a, 1)
	})
}

func TestHandlers_GetUserTags(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, user, _ := setupWithUsers(t, common3)

	for i := 0; i < 5; i++ {
		env.mustMakeTag(t, user.GetID(), rand)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/{userID}/tags", user.GetID().String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/{userID}/tags", user.GetID().String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(5)
	})
}

func TestHandlers_PatchUserTag(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, user, _ := setupWithUsers(t, common3)

	other := env.mustMakeUser(t, rand)
	tag := env.mustMakeTag(t, user.GetID(), rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusNoContent)

		ut, err := env.Repository.GetUserTag(user.GetID(), tag)
		require.NoError(t, err)
		assert.True(t, ut.GetIsLocked())
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(session.CookieName, env.generateSession(t, other.GetID())).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteUserTag(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, user, _ := setupWithUsers(t, common3)

	tag := env.mustMakeTag(t, user.GetID(), rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetUserTag(user.GetID(), tag)
		require.Equal(t, repository.ErrNotFound, err)
	})
}
