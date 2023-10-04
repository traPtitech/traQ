package v3

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func webhookEquals(t *testing.T, expect model.Webhook, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.GetID().String())
	actual.Value("botUserId").String().IsEqual(expect.GetBotUserID().String())
	actual.Value("displayName").String().IsEqual(expect.GetName())
	actual.Value("description").String().IsEqual(expect.GetDescription())
	actual.Value("secure").Boolean().IsEqual(len(expect.GetSecret()) > 0)
	actual.Value("channelId").String().IsEqual(expect.GetChannelID().String())
	actual.Value("ownerId").String().IsEqual(expect.GetCreatorID().String())
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
}

func TestHandlers_GetWebhooks(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks"
	env := Setup(t, s1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	ch := env.CreateChannel(t, rand)

	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)
	wh2 := env.CreateWebhook(t, rand, user2.GetID(), ch.ID)

	userSession := env.S(t, user.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (all=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, userSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		webhookEquals(t, wh, obj.Value(0).Object())
	})

	t.Run("success (all=false without permission)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, userSession).
			WithQuery("all", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		webhookEquals(t, wh, obj.Value(0).Object())
	})

	t.Run("success (all=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, adminSession).
			WithQuery("all", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(2)

		if obj.Value(0).Object().Value("id").Raw() == wh.GetID().String() {
			webhookEquals(t, wh, obj.Value(0).Object())
			webhookEquals(t, wh2, obj.Value(1).Object())
		} else {
			webhookEquals(t, wh2, obj.Value(0).Object())
			webhookEquals(t, wh, obj.Value(1).Object())
		}
	})
}

func TestHandlers_GetWebhookIcon(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}/icon"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)

	file, err := file2.GenerateIconFile(env.FM, "wh")
	require.NoError(t, err)
	wh, err := env.Repository.CreateWebhook(random2.AlphaNumeric(20), "", ch.ID, file, user.GetID(), "")
	require.NoError(t, err)

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, wh.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			HasContentType("image/png")
	})
}

func TestHandlers_CreateWebhook(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	s := env.S(t, user.GetID())

	req := &PostWebhooksRequest{
		Name:        random2.SecureAlphaNumeric(20),
		Description: "desc",
		ChannelID:   ch.ID,
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (nil uuid)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostWebhooksRequest{Name: "po", Description: "", ChannelID: uuid.Nil}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("botUserId").String().NotEmpty()
		obj.Value("displayName").String().IsEqual(req.Name)
		obj.Value("description").String().IsEqual(req.Description)
		obj.Value("secure").Boolean().IsFalse()
		obj.Value("channelId").String().IsEqual(req.ChannelID.String())
		obj.Value("ownerId").String().IsEqual(user.GetID().String())
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
	})
}

func TestHandlers_GetWebhook(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, wh.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		webhookEquals(t, wh, obj)
	})
}

func TestHandlers_EditWebhook(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, wh.GetID()).
			WithJSON(&PatchWebhookRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (empty name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchWebhookRequest{Name: optional.From("")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchWebhookRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchWebhookRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusNoContent)

		wh, err := env.Repository.GetWebhook(wh.GetID())
		require.NoError(t, err)
		assert.EqualValues(t, "po", wh.GetName())
	})
}

func TestHandlers_PostWebhook(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	ch2 := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	archived := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)

	calcHMACSHA1 := func(t *testing.T, message, secret string) string {
		t.Helper()
		mac := hmac.New(sha1.New, []byte(secret))
		_, _ = mac.Write([]byte(message))
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("bad request (empty body)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "", wh.GetSecret())).
			WithText("").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (no signature)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithText("test").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (bad signature)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "test", wh.GetSecret())).
			WithText("testTestTest").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (invalid channel id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "xxpoxx", wh.GetSecret())).
			WithHeader("X-TRAQ-Channel-id", "invalid").
			WithText("xxpoxx").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "xxpoxx", wh.GetSecret())).
			WithHeader("X-TRAQ-Channel-id", dm.ID.String()).
			WithText("xxpoxx").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (archived)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "xxpoxx", wh.GetSecret())).
			WithHeader("X-TRAQ-Channel-id", archived.ID.String()).
			WithText("xxpoxx").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "test", wh.GetSecret())).
			WithText("test").
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("unsupported media type", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "xxpoxx", wh.GetSecret())).
			WithJSON(map[string]interface{}{"text": "xxpoxx"}).
			Expect().
			Status(http.StatusUnsupportedMediaType)
	})

	t.Run("success with X-TRAQ-Channel-Id", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "xxpoxx", wh.GetSecret())).
			WithHeader("X-TRAQ-Channel-id", ch2.ID.String()).
			WithText("xxpoxx").
			Expect().
			Status(http.StatusNoContent)

		tl, err := env.MM.GetTimeline(message.TimelineQuery{Channel: ch2.ID})
		require.NoError(t, err)
		if assert.Len(t, tl.Records(), 1) {
			m := tl.Records()[0]
			assert.EqualValues(t, wh.GetBotUserID(), m.GetUserID())
			assert.EqualValues(t, "xxpoxx", m.GetText())
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetID()).
			WithHeader("X-TRAQ-Signature", calcHMACSHA1(t, "test", wh.GetSecret())).
			WithQuery("embed", 1).
			WithText("test").
			Expect().
			Status(http.StatusNoContent)

		tl, err := env.MM.GetTimeline(message.TimelineQuery{Channel: ch.ID})
		require.NoError(t, err)
		if assert.Len(t, tl.Records(), 1) {
			m := tl.Records()[0]
			assert.EqualValues(t, wh.GetBotUserID(), m.GetUserID())
			assert.EqualValues(t, "test", m.GetText())
		}
	})
}

func TestHandlers_DeleteWebhook(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)
	wh2 := env.CreateWebhook(t, rand, user2.GetID(), ch.ID)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, wh.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, wh2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetWebhook(wh.GetID())
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestHandlers_GetWebhookMessages(t *testing.T) {
	t.Parallel()

	path := "/api/v3/webhooks/{webhookId}/messages"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)
	m := env.CreateMessage(t, wh.GetBotUserID(), ch.ID, "test")
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, wh.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			WithQuery("limit", 500).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, wh.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		messageEquals(t, m, obj.Value(0).Object())
	})
}
