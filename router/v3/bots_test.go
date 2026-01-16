package v3

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/bot/event"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
)

func botEquals(t *testing.T, expect *model.Bot, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.ID.String())
	actual.Value("botUserId").String().IsEqual(expect.BotUserID.String())
	actual.Value("description").String().IsEqual(expect.Description)
	actual.Value("developerId").String().IsEqual(expect.CreatorID.String())
	actual.Value("subscribeEvents").Array().Length().IsEqual(len(expect.SubscribeEvents.Array()))
	actual.Value("mode").String().IsEqual(expect.Mode.String())
	actual.Value("state").Number().IsEqual(expect.State)
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
}

func TestHandlers_GetBots(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots"
	env := Setup(t, s1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	env.CreateBot(t, rand, user2.GetID())

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
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", false).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		botEquals(t, bot1, obj.Value(0).Object())
	})

	t.Run("success (all=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(2)
	})
}

func TestPostBotRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        string
		DisplayName string
		Description string
		Mode        string
		Endpoint    string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{Name: "", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"},
			true,
		},
		{
			"bad name",
			fields{Name: "ボットくん", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty display name",
			fields{Name: "name", DisplayName: "", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"},
			true,
		},
		{
			"bad display name",
			fields{Name: "name", DisplayName: strings.Repeat("a", 100), Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty desc",
			fields{Name: "name", DisplayName: "po", Description: "", Mode: "HTTP", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty mode",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "", Endpoint: "https://example.com"},
			true,
		},
		{
			"bad mode",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "bad mode", Endpoint: "https://example.com"},
			true,
		},
		{
			"should not require endpoint in WebSocket mode",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "WebSocket", Endpoint: ""},
			false,
		},
		{
			"endpoint is optional in WebSocket mode",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "WebSocket", Endpoint: "https://example.com"},
			false,
		},
		{
			"empty endpoint",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: ""},
			true,
		},
		{
			"bad endpoint (not url)",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "bad_url"},
			true,
		},
		{
			"bad endpoint (internal)",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://0.0.0.0:3000"},
			true,
		},
		{
			"success",
			fields{Name: "name", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostBotRequest{
				Name:        tt.fields.Name,
				DisplayName: tt.fields.DisplayName,
				Description: tt.fields.Description,
				Mode:        tt.fields.Mode,
				Endpoint:    tt.fields.Endpoint,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	commonSession := env.S(t, user.GetID())
	env.CreateBot(t, "575", user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "ボットくん", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "575", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "77", DisplayName: "po", Description: "desc", Mode: "HTTP", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("botUserId").String().NotEmpty()
		obj.Value("description").String().IsEqual("desc")
		obj.Value("subscribeEvents").Array().Length().IsEqual(0)
		obj.Value("mode").String().IsEqual("HTTP")
		obj.Value("state").Number().IsEqual(model.BotInactive)
		obj.Value("developerId").String().IsEqual(user.GetID().String())
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("tokens").Object().Value("verificationToken").String().NotEmpty()
		obj.Value("tokens").Object().Value("accessToken").String().NotEmpty()
		obj.Value("endpoint").String().IsEqual("https://example.com")
		obj.Value("privileged").Boolean().IsFalse()
		obj.Value("channels").Array().Length().IsEqual(0)
	})

	t.Run("success with WebSocket mode", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "78", DisplayName: "pop", Description: "desc", Mode: "WebSocket", Endpoint: ""}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("botUserId").String().NotEmpty()
		obj.Value("description").String().IsEqual("desc")
		obj.Value("subscribeEvents").Array().Length().IsEqual(0)
		obj.Value("mode").String().IsEqual("WebSocket")
		obj.Value("state").Number().IsEqual(model.BotActive)
		obj.Value("developerId").String().IsEqual(user.GetID().String())
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("tokens").Object().Value("verificationToken").String().NotEmpty()
		obj.Value("tokens").Object().Value("accessToken").String().NotEmpty()
		obj.Value("endpoint").String().IsEqual("")
		obj.Value("privileged").Boolean().IsFalse()
		obj.Value("channels").Array().Length().IsEqual(0)
	})
}

func TestHandlers_GetBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV7()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success 1 (detail=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		botEquals(t, bot1, obj)
	})

	t.Run("success 2 (detail=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		botEquals(t, bot2, obj)
	})

	t.Run("detail forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success (detail=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		botEquals(t, bot1, obj)
		obj.Value("tokens").Object().Value("verificationToken").String().NotEmpty()
		obj.Value("tokens").Object().Value("accessToken").String().NotEmpty()
		obj.Value("endpoint").String().IsEqual("https://example.com")
		obj.Value("privileged").Boolean().IsFalse()
		obj.Value("channels").Array().Length().IsEqual(0)
	})

	t.Run("success (detail=true, revoked=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		err := env.Repository.DeleteTokenByID(bot1.AccessTokenID)
		require.NoError(t, err)

		obj := e.GET(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		botEquals(t, bot1, obj)
		obj.Value("tokens").Object().Value("accessTokenRevoked").Boolean().IsTrue()
	})
}

func TestHandlers_EditBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user1.GetID())
	bot3 := env.CreateBot(t, rand, user2.GetID())
	wsBotWithEndpoint, err := env.Repository.CreateBot(random.AlphaNumeric(16), "po", "po", uuid.Nil, user1.GetID(), model.BotModeWebSocket, model.BotActive, "https://example.com")
	require.NoError(t, err)
	wsBotWithoutEndpoint, err := env.Repository.CreateBot(random.AlphaNumeric(16), "po", "po", uuid.Nil, user1.GetID(), model.BotModeWebSocket, model.BotActive, "")
	require.NoError(t, err)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (empty display name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DisplayName: optional.From("")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (too long display name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DisplayName: optional.From(strings.Repeat("a", 100))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (empty endpoint)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Endpoint: optional.From("")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (endpoint, not a url)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Endpoint: optional.From("po")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (endpoint, internal url)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Endpoint: optional.From("http://localhost:3000")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, nil id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.From(uuid.Nil)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, bad id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.From(bot1.ID)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, non existent id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.From(uuid.Must(uuid.NewV4()))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (subscribe events)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{SubscribeEvents: map[model.BotEventType]struct{}{
				"NON_EXISTENT_EVENT": {},
			}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (change mode to HTTP and endpoint not set)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, wsBotWithoutEndpoint.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Mode: optional.From("HTTP")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden (cannot patch others' bot)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot3.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (privileged)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Privileged: optional.From(true)}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Privileged: optional.From(true)}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (should be able to remove endpoint in WebSocket mode)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, wsBotWithEndpoint.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{
				Endpoint: optional.From(""),
			}).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{
				DisplayName: optional.From("po"),
				Description: optional.From("desc"),
				Mode:        optional.From("WebSocket"),
				Endpoint:    optional.From("https://example.com"),
				DeveloperID: optional.From(user2.GetID()),
				SubscribeEvents: map[model.BotEventType]struct{}{
					event.Ping: {},
				},
				Bio: optional.From("Bio"),
			}).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_DeleteBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_GetBotIcon(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/icon"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	commonSession := env.S(t, user.GetID())
	bot := env.CreateBot(t, rand, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK)
	})
}

func TestGetBotLogsRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Limit  int
		Offset int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"zero limit",
			fields{Limit: 0},
			false,
		},
		{
			"negative limit",
			fields{Limit: -1},
			true,
		},
		{
			"large limit",
			fields{Limit: 500},
			true,
		},
		{
			"negative offset",
			fields{Offset: -1},
			true,
		},
		{
			"success",
			fields{Limit: 50, Offset: 50},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GetBotLogsRequest{
				Limit:  tt.fields.Limit,
				Offset: tt.fields.Offset,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetBotLogs(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/logs"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	log := &model.BotEventLog{
		RequestID: uuid.Must(uuid.NewV4()),
		BotID:     bot1.ID,
		Event:     event.Ping,
		Result:    "ng",
		Code:      400,
		DateTime:  time.Now(),
	}
	require.NoError(t, env.Repository.WriteBotEventLog(log))

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (negative limit)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithQuery("limit", -1).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Keys().ContainsOnly(
			"botId", "requestId", "event", "result", "code", "datetime",
		)
		first.Value("botId").String().IsEqual(log.BotID.String())
		first.Value("requestId").String().IsEqual(log.RequestID.String())
		first.Value("event").String().IsEqual(log.Event.String())
		first.Value("result").String().IsEqual(log.Result)
		first.Value("code").Number().IsEqual(log.Code)
		first.Value("datetime").String().NotEmpty()
	})
}

func TestHandlers_GetChannelBots(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels/{channelId}/bots"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	commonSession := env.S(t, user.GetID())
	bot := env.CreateBot(t, rand, user.GetID())
	require.NoError(t, env.Repository.AddBotToChannel(bot.ID, channel.ID))

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("id").String().IsEqual(bot.ID.String())
		first.Value("botUserId").String().IsEqual(bot.BotUserID.String())
	})
}

func TestHandlers_ActivateBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/actions/activate"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusAccepted)
	})
}

func TestHandlers_InactivateBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/actions/inactivate"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_ReissueBot(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/actions/reissue"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("verificationToken").String().NotEmpty()
		obj.Value("accessToken").String().NotEmpty()
	})
}

func TestHandlers_LetBotJoinChannel(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/actions/join"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())
	channel := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user1.GetID(), user2.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel.ID}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (nil id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: uuid.Nil}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: dm.ID}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel.ID}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel.ID}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel.ID}).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_LetBotLeaveChannel(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots/{botId}/actions/leave"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())
	channel1 := env.CreateChannel(t, rand)
	channel2 := env.CreateChannel(t, rand)
	require.NoError(t, env.Repository.AddBotToChannel(bot1.ID, channel1.ID))

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel1.ID}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (nil id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: uuid.Nil}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel1.ID}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel1.ID}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (nop)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel2.ID}).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotActionJoinRequest{ChannelID: channel1.ID}).
			Expect().
			Status(http.StatusNoContent)
	})
}
