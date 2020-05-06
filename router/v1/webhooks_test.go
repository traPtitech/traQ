package v1

import (
	"encoding/hex"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/hmac"
	random2 "github.com/traPtitech/traQ/utils/random"
	"net/http"
	"strings"
	"testing"
)

func TestHandlers_GetWebhooks(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	for i := 0; i < 10; i++ {
		mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "")
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks").
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(10)
	})

	t.Run("Other user", func(t *testing.T) {
		t.Parallel()
		u := mustMakeUser(t, repo, rand)
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks").
			WithCookie(sessions.CookieName, generateSession(t, u.GetID())).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Empty()
	})
}

func TestHandlers_PostWebhooks(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common6)
	ch := mustMakeChannel(t, repo, rand)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks").
			WithJSON(map[string]string{"name": "test", "description": "test", "channelId": ch.ID.String()}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Bad Request (No channel)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks").
			WithJSON(map[string]string{"name": "test", "description": "test"}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		name := random2.AlphaNumeric(20)
		e := makeExp(t, server)
		id := e.POST("/api/1.0/webhooks").
			WithJSON(map[string]string{"name": name, "description": "test", "channelId": ch.ID.String()}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object().
			Value("webhookId").
			String().
			Raw()

		wb, err := repo.GetWebhook(uuid.FromStringOrNil(id))
		if assert.NoError(err) {
			assert.Equal(name, wb.GetName())
			assert.Equal("test", wb.GetDescription())
			assert.Equal(ch.ID, wb.GetChannelID())
			assert.Empty(wb.GetSecret())
		}
	})
}

func TestHandlers_GetWebhook(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "")

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}", wb.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Other user", func(t *testing.T) {
		t.Parallel()
		u := mustMakeUser(t, repo, rand)
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}", wb.GetID()).
			WithCookie(sessions.CookieName, generateSession(t, u.GetID())).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/webhooks/{webhookID}", wb.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()
		obj.Value("webhookId").String().Equal(wb.GetID().String())
		obj.Value("botUserId").String().Equal(wb.GetBotUserID().String())
		obj.Value("displayName").String().Equal(wb.GetName())
		obj.Value("description").String().Equal(wb.GetDescription())
		obj.Value("channelId").String().Equal(wb.GetChannelID().String())
		obj.Value("creatorId").String().Equal(wb.GetCreatorID().String())
	})
}

func TestHandlers_PatchWebhook(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "secret")

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookId}", uuid.Must(uuid.NewV4())).
			WithJSON(map[string]string{"name": strings.Repeat("a", 30)}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Other user", func(t *testing.T) {
		t.Parallel()
		u := mustMakeUser(t, repo, rand)
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookID}", wb.GetID()).
			WithJSON(map[string]string{"name": strings.Repeat("a", 30)}).
			WithCookie(sessions.CookieName, generateSession(t, u.GetID())).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Bad Request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithJSON(map[string]string{"name": strings.Repeat("a", 40)}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Channel Not found)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithJSON(map[string]uuid.UUID{"channelId": uuid.Must(uuid.NewV4())}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)
		name := random2.AlphaNumeric(20)
		desc := random2.AlphaNumeric(20)
		secret := random2.AlphaNumeric(20)
		ch := mustMakeChannel(t, repo, rand)
		e := makeExp(t, server)
		e.PATCH("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithJSON(map[string]string{"name": name, "description": desc, "channelId": ch.ID.String(), "secret": secret}).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		wb, err := repo.GetWebhook(wb.GetID())
		require.NoError(err)
		assert.Equal(name, wb.GetName())
		assert.Equal(desc, wb.GetDescription())
		assert.Equal(secret, wb.GetSecret())
		assert.Equal(ch.ID, wb.GetChannelID())
	})
}

func TestHandlers_DeleteWebhook(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "secret")

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/webhooks/{webhookId}", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Other user", func(t *testing.T) {
		t.Parallel()
		u := mustMakeUser(t, repo, rand)
		e := makeExp(t, server)
		e.DELETE("/api/1.0/webhooks/{webhookID}", wb.GetID()).
			WithCookie(sessions.CookieName, generateSession(t, u.GetID())).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "secret")
		e := makeExp(t, server)
		e.DELETE("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetWebhook(wb.GetID())
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}

func TestHandlers_PutWebhookIcon(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "secret")

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookId}/icon", wb.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookId}/icon", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Other user", func(t *testing.T) {
		t.Parallel()
		u := mustMakeUser(t, repo, rand)
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookID}/icon", wb.GetID()).
			WithCookie(sessions.CookieName, generateSession(t, u.GetID())).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Bad Request (No file)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookId}/icon", wb.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Not image)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookId}/icon", wb.GetID()).
			WithMultipart().
			WithFileBytes("file", "test.txt", []byte("text file")).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Bad image)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/api/1.0/webhooks/{webhookId}/icon", wb.GetID()).
			WithMultipart().
			WithFileBytes("file", "test.png", []byte("text file")).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusBadRequest)
	})
}

func TestHandlers_PostWebhook(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "secret")

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", uuid.Must(uuid.NewV4())).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("UnsupportedMediaType", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithJSON(map[string]string{"test": "test"}).
			Expect().
			Status(http.StatusUnsupportedMediaType)
	})

	t.Run("Bad Request (No Body)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Missing Signature)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("test").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("test").
			WithHeader(consts.HeaderSignature, "abcdef").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Bad Request (Nil Channel)", func(t *testing.T) {
		t.Parallel()
		body := "test"
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			WithHeader(consts.HeaderChannelID, "aaaa").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Channel not found)", func(t *testing.T) {
		t.Parallel()
		body := "test"
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			WithHeader(consts.HeaderChannelID, uuid.Must(uuid.NewV4()).String()).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Success1", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)
		body := "test"
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			Expect().
			Status(http.StatusNoContent)

		arr, _, err := repo.GetMessages(repository.MessagesQuery{Channel: ch.ID})
		require.NoError(err)
		if assert.Len(arr, 1) {
			assert.Equal(wb.GetBotUserID(), arr[0].UserID)
			assert.Equal(body, arr[0].Text)
		}
	})

	t.Run("Success2", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)
		body := "test"
		ch := mustMakeChannel(t, repo, rand)
		e := makeExp(t, server)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			WithHeader(consts.HeaderChannelID, ch.ID.String()).
			Expect().
			Status(http.StatusNoContent)

		arr, _, err := repo.GetMessages(repository.MessagesQuery{Channel: ch.ID})
		require.NoError(err)
		if assert.Len(arr, 1) {
			assert.Equal(wb.GetBotUserID(), arr[0].UserID)
			assert.Equal(body, arr[0].Text)
		}
	})
}

func TestHandlers_GetWebhookMessages(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, testUser, _ := setupWithUsers(t, common6)
	ch := mustMakeChannel(t, repo, rand)
	wb := mustMakeWebhook(t, repo, rand, ch.ID, testUser.GetID(), "")

	for i := 0; i < 60; i++ {
		mustMakeMessage(t, repo, wb.GetBotUserID(), ch.ID)
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}/messages", wb.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}/messages", wb.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			Length().
			Equal(60)
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}/messages", wb.GetID()).
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

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/webhooks/{webhookID}/messages", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})
}
