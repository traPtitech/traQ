package v1

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sessions"
	"net/http"
	"testing"
)

func TestHandlers_PutNotificationStatus(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common2)

	user := mustMakeUser(t, repo, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]string{"on": {user.GetID().String()}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		users := make([]uuid.UUID, 0)
		for _, subscription := range subscriptions {
			users = append(users, subscription.UserID)
		}

		assert.EqualValues(t, []uuid.UUID{user.GetID()}, users)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]uuid.UUID{"on": {uuid.Must(uuid.NewV4()), user.GetID(), uuid.Must(uuid.NewV4())}, "off": {uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		users := make([]uuid.UUID, 0)
		for _, subscription := range subscriptions {
			users = append(users, subscription.UserID)
		}

		assert.EqualValues(t, []uuid.UUID{user.GetID()}, users)
	})

	t.Run("Successful3", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannel(t, repo, rand)
		mustChangeChannelSubscription(t, repo, channel.ID, user.GetID())

		e := makeExp(t, server)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string][]string{"off": {user.GetID().String()}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		assert.Len(t, subscriptions, 0)
	})
}

func TestHandlers_GetNotificationStatus(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common2)

	channel := mustMakeChannel(t, repo, rand)
	user := mustMakeUser(t, repo, rand)

	mustChangeChannelSubscription(t, repo, channel.ID, user.GetID())

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

	user := mustMakeUser(t, repo, rand)
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, rand).ID, user.GetID())
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, rand).ID, user.GetID())

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/notification", user.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/{userID}/notification", user.GetID()).
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

	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, rand).ID, user.GetID())
	mustChangeChannelSubscription(t, repo, mustMakeChannel(t, repo, rand).ID, user.GetID())

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
