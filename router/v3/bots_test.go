package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect/v2"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_GetBots(t *testing.T) {
	t.Parallel()
	path := "/api/v3/bots"
	env := Setup(t, s1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	commonSession := env.S(t, user1.GetID())
	bot1 := env.CreateBot(t, rand, user1.GetID())
	env.CreateBot(t, rand, user2.GetID())

	botEquals := func(expect *model.Bot, actual *httpexpect.Object) {
		actual.Value("id").String().Equal(expect.ID.String())
		actual.Value("botUserId").String().Equal(expect.BotUserID.String())
		actual.Value("description").String().Equal(expect.Description)
		actual.Value("developerId").String().Equal(expect.CreatorID.String())
		actual.Value("subscribeEvents").Array().Length().Equal(len(expect.SubscribeEvents.Array()))
		actual.Value("state").Number().Equal(expect.State)
		actual.Value("createdAt").String().NotEmpty()
		actual.Value("updatedAt").String().NotEmpty()
	}

	t.Run("NotLoggedIn", func(t *testing.T) {
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

		botEquals(bot1, obj.Element(0).Object())
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

	t.Run("NotLoggedIn", func(t *testing.T) {
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
