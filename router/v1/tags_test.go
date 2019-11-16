package v1

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"
)

func TestHandlers_PostUserTag(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/users/{userID}/tags", user.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		tag := utils.RandAlphabetAndNumberString(20)
		e.POST("/api/1.0/users/{userID}/tags", user.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"tag": tag}).
			Expect().
			Status(http.StatusCreated)

		a, err := repo.GetUserIDsByTag(tag)
		require.NoError(t, err)
		assert.Len(t, a, 1)
		assert.Contains(t, a, user.ID)
	})
}

func TestHandlers_GetUserTags(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	for i := 0; i < 5; i++ {
		mustMakeTag(t, repo, user.ID, random)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/tags", user.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/tags", user.ID.String()).
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

	other := mustMakeUser(t, repo, random)
	tag := mustMakeTag(t, repo, user.ID, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID.String(), tag.String()).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID.String(), tag.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusNoContent)

		ut, err := repo.GetUserTag(user.ID, tag)
		require.NoError(t, err)
		assert.True(t, ut.IsLocked)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/users/{userID}/tags/{tagID}", user.ID.String(), tag.String()).
			WithCookie(sessions.CookieName, generateSession(t, other.ID)).
			WithJSON(map[string]bool{"isLocked": true}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteUserTag(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common3)

	tag := mustMakeTag(t, repo, user.ID, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.ID.String(), tag.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/{userID}/tags/{tagID}", user.ID.String(), tag.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetUserTag(user.ID, tag)
		require.Equal(t, repository.ErrNotFound, err)
	})
}
