package v1

import (
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/utils/hmac"
)

func TestHandlers_PostWebhook(t *testing.T) {
	t.Parallel()
	env, _, _, _, _, testUser, _ := setupWithUsers(t, common6)
	ch := env.mustMakeChannel(t, rand)
	wb := env.mustMakeWebhook(t, rand, ch.ID, testUser.GetID(), "secret")

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", uuid.Must(uuid.NewV4())).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("UnsupportedMediaType", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithJSON(map[string]string{"test": "test"}).
			Expect().
			Status(http.StatusUnsupportedMediaType)
	})

	t.Run("Bad Request (No Body)", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Missing Signature)", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("test").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText("test").
			WithHeader(consts.HeaderSignature, "abcdef").
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Bad Request (Nil Channel)", func(t *testing.T) {
		t.Parallel()
		body := "test"
		e := env.makeExp(t)
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
		e := env.makeExp(t)
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
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			Expect().
			Status(http.StatusNoContent)

		arr, _, err := env.Repository.GetMessages(repository.MessagesQuery{Channel: ch.ID})
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
		ch := env.mustMakeChannel(t, rand)
		e := env.makeExp(t)
		e.POST("/api/1.0/webhooks/{webhookId}", wb.GetID()).
			WithText(body).
			WithHeader(consts.HeaderSignature, hex.EncodeToString(hmac.SHA1([]byte(body), wb.GetSecret()))).
			WithHeader(consts.HeaderChannelID, ch.ID.String()).
			Expect().
			Status(http.StatusNoContent)

		arr, _, err := env.Repository.GetMessages(repository.MessagesQuery{Channel: ch.ID})
		require.NoError(err)
		if assert.Len(arr, 1) {
			assert.Equal(wb.GetBotUserID(), arr[0].UserID)
			assert.Equal(body, arr[0].Text)
		}
	})
}
