package v1

import (
	"net/http"
	"testing"

	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_DeleteUnread(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)
	message := env.mustMakeMessage(t, testUser.GetID(), channel.ID)
	env.mustMakeMessageUnread(t, testUser.GetID(), message.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/me/unread/channels/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/users/me/unread/channels/{channelID}", channel.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})
}
