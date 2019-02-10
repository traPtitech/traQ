package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGroup_Messages(t *testing.T) {
	assert, require, session, _ := beforeTest(t)

	t.Run("TestGetMessageByID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		postmanID := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).ID
		privateID := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{postmanID}).ID
		message2 := mustMakeMessage(t, postmanID, privateID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/messages/{messageID}", message.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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
			e := makeExp(t)

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
			e := makeExp(t)
			e.GET("/api/1.0/messages/{messageID}", message2.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNotFound)
		})
	})

	t.Run("TestPostMessage", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		postmanID := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).ID
		privateID := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{postmanID}).ID

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
				WithJSON(map[string]string{"text": "test message"}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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

			_, err := model.GetMessageByID(uuid.FromStringOrNil(obj.Value("messageId").String().Raw()))
			require.NoError(err)
		})

		t.Run("Successful2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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

			_, err := model.GetMessageByID(uuid.FromStringOrNil(obj.Value("messageId").String().Raw()))
			require.NoError(err)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/channels/{channelID}/messages", privateID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"text": "test message"}).
				Expect().
				Status(http.StatusNotFound)
		})

		t.Run("Failure2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"not_text_field": "not_text_field"}).
				Expect().
				Status(http.StatusBadRequest)
		})
	})

	t.Run("TestGetMessagesByChannelID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		postmanID := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).ID
		privateID := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{postmanID}).ID

		for i := 0; i < 5; i++ {
			mustMakeMessage(t, testUser.ID, channel.ID)
		}

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/messages", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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
			e := makeExp(t)
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
			e := makeExp(t)
			e.GET("/api/1.0/channels/{channelID}/messages", privateID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNotFound)
		})
	})

	t.Run("TestPutMessageByID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		postmanID := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).ID
		privateID := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{postmanID}).ID
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		message2 := mustMakeMessage(t, postmanID, privateID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
				WithJSON(map[string]string{"text": "new message"}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			text := "new message"
			e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"text": text}).
				Expect().
				Status(http.StatusNoContent)

			m, err := model.GetMessageByID(message.ID)
			require.NoError(err)
			assert.Equal(text, m.Text)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/messages/{messageID}", message2.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNotFound)
		})

		t.Run("Failure2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.PUT("/api/1.0/messages/{messageID}", message.ID.String()).
				WithCookie(sessions.CookieName, generateSession(t, postmanID)).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestDeleteMessageByID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		postmanID := mustCreateUser(t, utils.RandAlphabetAndNumberString(20)).ID
		privateID := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{postmanID}).ID
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		message2 := mustMakeMessage(t, postmanID, privateID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNoContent)

			_, err := model.GetMessageByID(message.ID)
			require.Equal(model.ErrNotFound, err)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/messages/{messageID}", message2.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNotFound)
		})

		t.Run("Failure2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			message := mustMakeMessage(t, testUser.ID, channel.ID)
			e.DELETE("/api/1.0/messages/{messageID}", message.ID.String()).
				WithCookie(sessions.CookieName, generateSession(t, postmanID)).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestPostMessageReport", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
				WithJSON(map[string]string{"reason": "aaaa"}).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"reason": "aaaa"}).
				Expect().
				Status(http.StatusNoContent)

			r, err := model.GetMessageReportsByMessageID(message.ID)
			require.NoError(err)
			assert.Len(r, 1)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/messages/{messageID}/report", message.ID.String()).
				WithCookie(sessions.CookieName, session).
				WithJSON(map[string]string{"not_reason": "aaaa"}).
				Expect().
				Status(http.StatusBadRequest)
		})
	})

	t.Run("TestGetUnread", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		user := mustCreateUser(t, utils.RandAlphabetAndNumberString(20))
		mustMakeUnread(t, user.ID, message.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me/unread").
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/users/me/unread").
				WithCookie(sessions.CookieName, generateSession(t, user.ID)).
				Expect().
				Status(http.StatusOK).
				JSON().
				Array().
				Length().
				Equal(1)
		})
	})

	t.Run("TestDeleteUnread", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, testUser.ID, utils.RandAlphabetAndNumberString(20), "")
		message := mustMakeMessage(t, testUser.ID, channel.ID)
		mustMakeUnread(t, testUser.ID, message.ID)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/me/unread/{channelID}", channel.ID.String()).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/users/me/unread/{channelID}", channel.ID.String()).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusNoContent)
		})
	})
}
