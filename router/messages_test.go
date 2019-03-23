package router

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"testing"
)

func TestHandlers_GetMessageByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	postmanID := mustMakeUser(t, repo, random).ID
	privateID := mustMakePrivateChannel(t, repo, random, []uuid.UUID{postmanID}).ID
	message2 := mustMakeMessage(t, repo, postmanID, privateID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/messages/{messageID}", message.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("messageId").String().Equal(message.ID.String())
		obj.Value("userId").String().Equal(testUser.ID.String())
		obj.Value("parentChannelId").String().Equal(channel.ID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message.Text)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)

		obj := e.GET("/api/1.0/messages/{messageID}", message2.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, postmanID)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("messageId").String().Equal(message2.ID.String())
		obj.Value("userId").String().Equal(postmanID.String())
		obj.Value("parentChannelId").String().Equal(privateID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message2.Text)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/messages/{messageID}", message2.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})
}

func TestHandlers_PostMessage(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	postmanID := mustMakeUser(t, repo, random).ID
	privateID := mustMakePrivateChannel(t, repo, random, []uuid.UUID{postmanID}).ID

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithJSON(map[string]string{"text": "test message"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		message := "test message"

		obj := e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"text": message}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("messageId").String().NotEmpty()
		obj.Value("userId").String().Equal(testUser.ID.String())
		obj.Value("parentChannelId").String().Equal(channel.ID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()

		_, err := repo.GetMessageByID(uuid.FromStringOrNil(obj.Value("messageId").String().Raw()))
		assert.NoError(t, err)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		message := "test message"

		obj := e.POST("/api/1.0/channels/{channelID}/messages", privateID).
			WithCookie(sessions.CookieName, generateSession(t, postmanID)).
			WithJSON(map[string]string{"text": message}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("messageId").String().NotEmpty()
		obj.Value("userId").String().Equal(postmanID.String())
		obj.Value("parentChannelId").String().Equal(privateID.String())
		obj.Value("pin").Boolean().False()
		obj.Value("content").String().Equal(message)
		obj.Value("reported").Boolean().False()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("stampList").Array().Empty()

		_, err := repo.GetMessageByID(uuid.FromStringOrNil(obj.Value("messageId").String().Raw()))
		assert.NoError(t, err)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels/{channelID}/messages", privateID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"text": "test message"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Failure2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"not_text_field": "not_text_field"}).
			Expect().
			Status(http.StatusBadRequest)
	})
}

func TestHandlers_GetMessagesByChannelID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	postmanID := mustMakeUser(t, repo, random).ID
	privateID := mustMakePrivateChannel(t, repo, random, []uuid.UUID{postmanID}).ID

	for i := 0; i < 5; i++ {
		mustMakeMessage(t, repo, testUser.ID, channel.ID)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(5)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
			WithQuery("limit", 3).
			WithQuery("offset", 1).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(3)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/channels/{channelID}/messages", privateID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})
}

func TestHandlers_PutMessageByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	postmanID := mustMakeUser(t, repo, random).ID
	privateID := mustMakePrivateChannel(t, repo, random, []uuid.UUID{postmanID}).ID
	message2 := mustMakeMessage(t, repo, postmanID, privateID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithJSON(map[string]string{"text": "new message"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		text := "new message"
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"text": text}).
			Expect().
			Status(http.StatusNoContent)

		m, err := repo.GetMessageByID(message.ID)
		assert.NoError(t, err)
		assert.Equal(t, text, m.Text)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/messages/{messageID}", message2.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Failure2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, postmanID)).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_DeleteMessageByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	postmanID := mustMakeUser(t, repo, random).ID
	privateID := mustMakePrivateChannel(t, repo, random, []uuid.UUID{postmanID}).ID
	message2 := mustMakeMessage(t, repo, postmanID, privateID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetMessageByID(message.ID)
		assert.Equal(t, repository.ErrNotFound, err)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/messages/{messageID}", message2.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Failure2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
		e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
			WithCookie(sessions.CookieName, generateSession(t, postmanID)).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_PostMessageReport(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
			WithJSON(map[string]string{"reason": "aaaa"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"reason": "aaaa"}).
			Expect().
			Status(http.StatusNoContent)

		r, err := repo.GetMessageReportsByMessageID(message.ID)
		assert.NoError(t, err)
		assert.Len(t, r, 1)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
			WithCookie(sessions.CookieName, session).
			WithJSON(map[string]string{"not_reason": "aaaa"}).
			Expect().
			Status(http.StatusBadRequest)
	})
}

func TestHandlers_GetUnread(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	user := mustMakeUser(t, repo, random)
	mustMakeMessageUnread(t, repo, user.ID, message.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/unread").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/users/me/unread").
			WithCookie(sessions.CookieName, generateSession(t, user.ID)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(1)
	})
}

func TestHandlers_DeleteUnread(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common2)

	channel := mustMakeChannel(t, repo, random)
	message := mustMakeMessage(t, repo, testUser.ID, channel.ID)
	mustMakeMessageUnread(t, repo, testUser.ID, message.ID)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/me/unread/{channelID}", channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/users/me/unread/{channelID}", channel.ID.String()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)
	})
}
