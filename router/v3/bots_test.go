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
)

func botEquals(t *testing.T, expect *model.Bot, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().Equal(expect.ID.String())
	actual.Value("botUserId").String().Equal(expect.BotUserID.String())
	actual.Value("description").String().Equal(expect.Description)
	actual.Value("developerId").String().Equal(expect.CreatorID.String())
	actual.Value("subscribeEvents").Array().Length().Equal(len(expect.SubscribeEvents.Array()))
	actual.Value("state").Number().Equal(expect.State)
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

		obj.Length().Equal(1)

		botEquals(t, bot1, obj.Element(0).Object())
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

		obj.Length().Equal(2)
	})
}

func TestPostBotRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        string
		DisplayName string
		Description string
		Endpoint    string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{Name: "", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"},
			true,
		},
		{
			"bad name",
			fields{Name: "ボットくん", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty display name",
			fields{Name: "name", DisplayName: "", Description: "desc", Endpoint: "https://example.com"},
			true,
		},
		{
			"bad display name",
			fields{Name: "name", DisplayName: strings.Repeat("a", 100), Description: "desc", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty desc",
			fields{Name: "name", DisplayName: "po", Description: "", Endpoint: "https://example.com"},
			true,
		},
		{
			"empty endpoint",
			fields{Name: "name", DisplayName: "po", Description: "desc", Endpoint: ""},
			true,
		},
		{
			"bad endpoint (not url)",
			fields{Name: "name", DisplayName: "po", Description: "desc", Endpoint: "bad_url"},
			true,
		},
		{
			"bad endpoint (internal)",
			fields{Name: "name", DisplayName: "po", Description: "desc", Endpoint: "https://0.0.0.0:3000"},
			true,
		},
		{
			"success",
			fields{Name: "name", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostBotRequest{
				Name:        tt.fields.Name,
				DisplayName: tt.fields.DisplayName,
				Description: tt.fields.Description,
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
			WithJSON(&PostBotRequest{Name: "ボットくん", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "575", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostBotRequest{Name: "77", DisplayName: "po", Description: "desc", Endpoint: "https://example.com"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("botUserId").String().NotEmpty()
		obj.Value("description").String().Equal("desc")
		obj.Value("subscribeEvents").Array().Length().Equal(0)
		obj.Value("state").Number().Equal(model.BotInactive)
		obj.Value("developerId").String().Equal(user.GetID().String())
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("tokens").Object().Value("verificationToken").String().NotEmpty()
		obj.Value("tokens").Object().Value("accessToken").String().NotEmpty()
		obj.Value("endpoint").String().Equal("https://example.com")
		obj.Value("privileged").Boolean().False()
		obj.Value("channels").Array().Length().Equal(0)
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

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
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
		obj.Value("endpoint").String().Equal("https://example.com")
		obj.Value("privileged").Boolean().False()
		obj.Value("channels").Array().Length().Equal(0)
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

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (display name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DisplayName: optional.StringFrom(strings.Repeat("a", 100))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (endpoint, not a url)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Endpoint: optional.StringFrom("po")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (endpoint, internal url)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Endpoint: optional.StringFrom("http://localhost:3000")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, nil id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.UUIDFrom(uuid.Nil)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, bad id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.UUIDFrom(bot1.ID)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (developer id, non existent id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DeveloperID: optional.UUIDFrom(uuid.Must(uuid.NewV4()))}).
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

	t.Run("forbidden (cannot patch others' bot)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot3.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{DisplayName: optional.StringFrom("po")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (privileged)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot1.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Privileged: optional.BoolFrom(true)}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{Privileged: optional.BoolFrom(true)}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, bot2.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PatchBotRequest{
				DisplayName: optional.StringFrom("po"),
				Description: optional.StringFrom("desc"),
				Endpoint:    optional.StringFrom("https://example.com"),
				DeveloperID: optional.UUIDFrom(user2.GetID()),
				SubscribeEvents: map[model.BotEventType]struct{}{
					event.Ping: {},
				},
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

		obj.Length().Equal(1)

		first := obj.First().Object()
		first.Value("botId").String().Equal(log.BotID.String())
		first.Value("requestId").String().Equal(log.RequestID.String())
		first.Value("event").String().Equal(log.Event.String())
		first.Value("code").Number().Equal(log.Code)
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

		obj.Length().Equal(1)

		first := obj.Element(0).Object()
		first.Value("id").String().Equal(bot.ID.String())
		first.Value("botUserId").String().Equal(bot.BotUserID.String())
	})
}

func TestHandlers_ActivateBot(t *testing.T) {
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
