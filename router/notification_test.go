package router

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"testing"
)

func TestHandlers_PutNotificationStatus(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common2)

	user := mustMakeUser(t, repo, random)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, random)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, random)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]string{"on": {user.ID.String()}}).
			Expect().
			Status(http.StatusNoContent)

		users, err := repo.GetSubscribingUserIDs(channel.ID)
		require.NoError(t, err)
		assert.EqualValues(t, []uuid.UUID{user.ID}, users)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, random)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]uuid.UUID{"on": {uuid.Must(uuid.NewV4()), user.ID, uuid.Must(uuid.NewV4())}, "off": {uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusNoContent)

		users, err := repo.GetSubscribingUserIDs(channel.ID)
		require.NoError(t, err)
		assert.EqualValues(t, []uuid.UUID{user.ID}, users)
	})

	t.Run("Successful3", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, random)
		mustChangeChannelSubscription(t, repo, channel.ID, user.ID, true)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]string{"off": {user.ID.String()}}).
			Expect().
			Status(http.StatusNoContent)

		users, err := repo.GetSubscribingUserIDs(channel.ID)
		require.NoError(t, err)
		assert.Len(t, users, 0)
	})
}

func TestHandlers_GetNotificationStatus(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common2)

	channel := mustMakeChannel(t, repo, random)
	user := mustMakeUser(t, repo, random)

	mustChangeChannelSubscription(t, repo, channel.ID, user.ID, true)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(1)
	})
}

func TestHandlers_GetNotificationChannels(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common2)

	user := mustMakeUser(t, repo, random)
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, random).ID, user.ID, true)
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, random).ID, user.ID, true)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/notification", user.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/notification", user.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(2)
	})
}

func TestHandlers_GetMyNotificationChannels(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common2)

	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, random).ID, user.ID, true)
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, random).ID, user.ID, true)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/notification").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/notification").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(2)
	})
}
