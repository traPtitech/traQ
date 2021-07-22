package v1

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_PutNotificationStatus(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common2)

	user := env.mustMakeUser(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()

		channel := env.mustMakeChannel(t, rand)

		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()

		channel := env.mustMakeChannel(t, rand)

		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string][]string{"on": {user.GetID().String()}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		users := make([]uuid.UUID, 0)
		for _, subscription := range subscriptions {
			users = append(users, subscription.UserID)
		}

		assert.EqualValues(t, []uuid.UUID{user.GetID()}, users)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()

		channel := env.mustMakeChannel(t, rand)

		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string][]uuid.UUID{"on": {uuid.Must(uuid.NewV4()), user.GetID(), uuid.Must(uuid.NewV4())}, "off": {uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		users := make([]uuid.UUID, 0)
		for _, subscription := range subscriptions {
			users = append(users, subscription.UserID)
		}

		assert.EqualValues(t, []uuid.UUID{user.GetID()}, users)
	})

	t.Run("Successful3", func(t *testing.T) {
		t.Parallel()

		channel := env.mustMakeChannel(t, rand)
		env.mustChangeChannelSubscription(t, channel.ID, user.GetID())

		e := env.makeExp(t)
		e.PUT("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string][]string{"off": {user.GetID().String()}}).
			Expect().
			Status(http.StatusNoContent)

		subscriptions, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetChannel(channel.ID).SetLevel(model.ChannelSubscribeLevelMarkAndNotify))
		require.NoError(t, err)
		assert.Len(t, subscriptions, 0)
	})
}

func TestHandlers_GetNotificationStatus(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common2)

	channel := env.mustMakeChannel(t, rand)
	user := env.mustMakeUser(t, rand)

	env.mustChangeChannelSubscription(t, channel.ID, user.GetID())

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/notification", channel.ID.String()).
			WithCookie(session.CookieName, s).
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
	env, _, _, s, _ := setup(t, common2)

	user := env.mustMakeUser(t, rand)
	env.mustChangeChannelSubscription(t, env.mustMakeChannel(t, rand).ID, user.GetID())
	env.mustChangeChannelSubscription(t, env.mustMakeChannel(t, rand).ID, user.GetID())

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/{userID}/notification", user.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/{userID}/notification", user.GetID()).
			WithCookie(session.CookieName, s).
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
	env, _, _, s, _, user, _ := setupWithUsers(t, common2)

	env.mustChangeChannelSubscription(t, env.mustMakeChannel(t, rand).ID, user.GetID())
	env.mustChangeChannelSubscription(t, env.mustMakeChannel(t, rand).ID, user.GetID())

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/me/notification").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/users/me/notification").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(2)
	})
}
