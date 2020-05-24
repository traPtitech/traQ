package v1

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"net/http"
	"testing"
)

func TestHandlers_GetMessageByID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)
	message := env.mustMakeMessage(t, testUser.GetID(), channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/messages/{messageID}", message.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		obj := e.GET("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("messageId").String().Equal(message.ID.String())
		obj.Value("userId").String().Equal(testUser.GetID().String())
		obj.Value("parentChannelId").String().Equal(channel.ID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message.Text)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()
	})
}

func TestHandlers_PostMessage(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithJSON(map[string]string{"text": "test message"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		message := "test message"

		obj := e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"text": message}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("messageId").String().NotEmpty()
		obj.Value("userId").String().Equal(testUser.GetID().String())
		obj.Value("parentChannelId").String().Equal(channel.ID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()

		_, err := env.Repository.GetMessageByID(uuid.FromStringOrNil(obj.Value("messageId").String().Raw()))
		assert.NoError(t, err)
	})

	t.Run("Failure2", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"not_text_field": "not_text_field"}).
			Expect().
			Status(http.StatusBadRequest)
	})
}

func TestHandlers_GetMessagesByChannelID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)

	for i := 0; i < 5; i++ {
		env.mustMakeMessage(t, testUser.GetID(), channel.ID)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(5)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithQuery("limit", 3).
			WithQuery("offset", 1).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(3)
	})
}

func TestHandlers_PutMessageByID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)
	message := env.mustMakeMessage(t, testUser.GetID(), channel.ID)
	postmanID := env.mustMakeUser(t, rand).GetID()

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithJSON(map[string]string{"text": "new message"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		text := "new message"
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"text": text}).
			Expect().
			Status(http.StatusNoContent)

		m, err := env.Repository.GetMessageByID(message.ID)
		assert.NoError(t, err)
		assert.Equal(t, text, m.Text)
	})

	t.Run("Failure2", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, env.generateSession(t, postmanID)).
			WithJSON(map[string]string{"text": "new message"}).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteMessageByID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _, testUser, _ := setupWithUsers(t, common2)

	channel := env.mustMakeChannel(t, rand)
	message := env.mustMakeMessage(t, testUser.GetID(), channel.ID)
	postmanID := env.mustMakeUser(t, rand).GetID()

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetMessageByID(message.ID)
		assert.Equal(t, repository.ErrNotFound, err)
	})

	t.Run("Webhook Message", func(t *testing.T) {
		t.Parallel()
		wb := env.mustMakeWebhook(t, rand, channel.ID, testUser.GetID(), "")
		message := env.mustMakeMessage(t, wb.GetBotUserID(), channel.ID)

		e := env.makeExp(t)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetMessageByID(message.ID)
		assert.Equal(t, repository.ErrNotFound, err)
	})

	t.Run("Forbidden (other's message)", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		message := env.mustMakeMessage(t, testUser.GetID(), channel.ID)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, env.generateSession(t, postmanID)).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Forbidden (other's webhook message)", func(t *testing.T) {
		t.Parallel()
		wb := env.mustMakeWebhook(t, rand, channel.ID, testUser.GetID(), "")
		message := env.mustMakeMessage(t, wb.GetBotUserID(), channel.ID)

		e := env.makeExp(t)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(session.CookieName, env.generateSession(t, postmanID)).
			Expect().
			Status(http.StatusForbidden)
	})
}

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
