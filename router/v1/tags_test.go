package v1

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sessions"
	random2 "github.com/traPtitech/traQ/utils/random"
	"net/http"
	"testing"
)

func TestHandlers_PostUserTag(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/users/{userID}/tags", user.GetID().String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		tag := random2.AlphaNumeric(20)
		e.POST("/api/1.0/users/{userID}/tags", user.GetID().String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"tag": tag}).
			Expect().
			Status(http.StatusCreated)

		a, err := repo.GetUserIDsByTag(tag)
		require.NoError(t, err)
		assert.Len(t, a, 1)
		assert.Contains(t, a, user.GetID())
	})
}

func TestHandlers_GetUserTags(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	for i := 0; i < 5; i++ {
		mustMakeTag(t, repo, user.GetID(), rand)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/tags", user.GetID().String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/tags", user.GetID().String()).
			WithCookie(sessions.CookieName, session).
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
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	other := mustMakeUser(t, repo, rand)
	tag := mustMakeTag(t, repo, user.GetID(), rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusNoContent)

		ut, err := repo.GetUserTag(user.GetID(), tag)
		require.NoError(t, err)
		assert.True(t, ut.IsLocked)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(sessions.CookieName, generateSession(t, other.GetID())).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteUserTag(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	tag := mustMakeTag(t, repo, user.GetID(), rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.GetID().String(), tag.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetUserTag(user.GetID(), tag)
		require.Equal(t, repository.ErrNotFound, err)
	})
}
