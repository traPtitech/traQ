package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/message"
)

func TestHandlers_GetMyUnreadChannels(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/unread"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	env.MakeMessageUnread(t, user.GetID(), m.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().Equal(1)

		first := obj.First().Object()
		first.Value("channelId").String().Equal(ch.ID.String())
		first.Value("count").Number().Equal(1)
		first.Value("noticeable").Boolean().False()
		first.Value("since").String().NotEmpty()
		first.Value("updatedAt").String().NotEmpty()
	})
}

func TestHandlers_ReadChannel(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/unread/{channelId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	env.MakeMessageUnread(t, user.GetID(), m.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ch.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ch.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		chs, err := env.Repository.GetUserUnreadChannels(user.GetID())
		require.NoError(t, err)
		assert.Len(t, chs, 0)
	})
}

func TestHandlers_GetMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	dmm := env.CreateMessage(t, user2.GetID(), dm.ID, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, m.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, dmm.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
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
		obj := e.GET(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		messageEquals(t, m, obj)
	})
}

func TestPostMessageRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Content string
		Embed   bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty content",
			fields{Content: ""},
			true,
		},
		{
			"too long content",
			fields{Content: strings.Repeat("a", 12000)},
			true,
		},
		{
			"success",
			fields{Content: "Hello, traP"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostMessageRequest{
				Content: tt.fields.Content,
				Embed:   tt.fields.Embed,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}"
	env := Setup(t, common1)

	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)

	pub := env.CreateChannel(t, rand)

	bot := env.CreateBot(t, rand, user.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())
	bot3 := env.CreateBot(t, rand, user.GetID())
	w := env.CreateWebhook(t, rand, user.GetID(), pub.ID)
	w2 := env.CreateWebhook(t, rand, user2.GetID(), pub.ID)
	w3 := env.CreateWebhook(t, rand, user.GetID(), pub.ID)

	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	dm2 := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	dm3 := env.CreateDMChannel(t, user2.GetID(), bot.BotUserID)
	archived := env.CreateChannel(t, rand)

	m := env.CreateMessage(t, user.GetID(), pub.ID, rand)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))

	bot3M := env.CreateMessage(t, bot3.BotUserID, pub.ID, rand)
	w3M := env.CreateMessage(t, w3.GetBotUserID(), pub.ID, rand)
	require.NoError(t, env.Repository.DeleteBot(bot3.ID))
	require.NoError(t, env.Repository.DeleteWebhook(w3.GetID()))

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, m.GetID()).
			WithJSON(&PostMessageRequest{Content: "po"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, m.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMessageRequest{Content: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMessageRequest{Content: "po"}).
			Expect().
			Status(http.StatusNotFound)
	})

	tests := []struct {
		name   string
		m      message.Message
		expect int
	}{
		{
			"self message (public)",
			m,
			http.StatusNoContent,
		},
		{
			"archived message",
			archivedM,
			http.StatusBadRequest,
		},
		{
			"other's message (public)",
			env.CreateMessage(t, user3.GetID(), pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"self message (dm)",
			env.CreateMessage(t, user.GetID(), dm.ID, rand),
			http.StatusNoContent,
		},
		{
			"other's message (dm)",
			env.CreateMessage(t, user2.GetID(), dm.ID, rand),
			http.StatusForbidden,
		},
		{
			"other's dm",
			env.CreateMessage(t, user2.GetID(), dm2.ID, rand),
			http.StatusNotFound,
		},
		{
			"my bot message (public)",
			env.CreateMessage(t, bot.BotUserID, pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"other's bot message (public)",
			env.CreateMessage(t, bot2.BotUserID, pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"my bot message (dm)",
			env.CreateMessage(t, bot.BotUserID, dm3.ID, rand),
			http.StatusNotFound,
		},
		{
			"my deleted bot message (public)",
			bot3M,
			http.StatusForbidden,
		},
		{
			"my webhook message (public)",
			env.CreateMessage(t, w.GetBotUserID(), pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"other's webhook message (public)",
			env.CreateMessage(t, w2.GetBotUserID(), pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"my deleted webhook message (public)",
			w3M,
			http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := env.R(t)
			e.PUT(path, tt.m.GetID()).
				WithCookie(session.CookieName, s).
				WithJSON(&PostMessageRequest{Content: "po", Embed: true}).
				Expect().
				Status(tt.expect)
		})
	}
}

func TestHandlers_DeleteMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}"
	env := Setup(t, common1)

	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)

	pub := env.CreateChannel(t, rand)

	bot := env.CreateBot(t, rand, user.GetID())
	bot2 := env.CreateBot(t, rand, user2.GetID())
	bot3 := env.CreateBot(t, rand, user.GetID())
	w := env.CreateWebhook(t, rand, user.GetID(), pub.ID)
	w2 := env.CreateWebhook(t, rand, user2.GetID(), pub.ID)
	w3 := env.CreateWebhook(t, rand, user.GetID(), pub.ID)

	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	dm2 := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	dm3 := env.CreateDMChannel(t, user2.GetID(), bot.BotUserID)
	archived := env.CreateChannel(t, rand)

	m := env.CreateMessage(t, user.GetID(), pub.ID, rand)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))

	bot3M := env.CreateMessage(t, bot3.BotUserID, pub.ID, rand)
	w3M := env.CreateMessage(t, w3.GetBotUserID(), pub.ID, rand)
	require.NoError(t, env.Repository.DeleteBot(bot3.ID))
	require.NoError(t, env.Repository.DeleteWebhook(w3.GetID()))

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	tests := []struct {
		name   string
		m      message.Message
		expect int
	}{
		{
			"self message (public)",
			m,
			http.StatusNoContent,
		},
		{
			"archived message",
			archivedM,
			http.StatusBadRequest,
		},
		{
			"other's message (public)",
			env.CreateMessage(t, user3.GetID(), pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"self message (dm)",
			env.CreateMessage(t, user.GetID(), dm.ID, rand),
			http.StatusNoContent,
		},
		{
			"other's message (dm)",
			env.CreateMessage(t, user2.GetID(), dm.ID, rand),
			http.StatusForbidden,
		},
		{
			"other's dm",
			env.CreateMessage(t, user2.GetID(), dm2.ID, rand),
			http.StatusNotFound,
		},
		{
			"my bot message (public)",
			env.CreateMessage(t, bot.BotUserID, pub.ID, rand),
			http.StatusNoContent,
		},
		{
			"other's bot message (public)",
			env.CreateMessage(t, bot2.BotUserID, pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"my bot message (dm)",
			env.CreateMessage(t, bot.BotUserID, dm3.ID, rand),
			http.StatusNotFound,
		},
		{
			"my deleted bot message (public)",
			bot3M,
			http.StatusForbidden,
		},
		{
			"my webhook message (public)",
			env.CreateMessage(t, w.GetBotUserID(), pub.ID, rand),
			http.StatusNoContent,
		},
		{
			"other's webhook message (public)",
			env.CreateMessage(t, w2.GetBotUserID(), pub.ID, rand),
			http.StatusForbidden,
		},
		{
			"my deleted webhook message (public)",
			w3M,
			http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := env.R(t)
			e.DELETE(path, tt.m.GetID()).
				WithCookie(session.CookieName, s).
				Expect().
				Status(tt.expect)
		})
	}
}

func TestHandlers_GetPin(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/pin"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	m2 := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	_, err := env.MM.Pin(m.GetID(), user.GetID())
	require.NoError(t, err)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, m.GetID()).
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

	t.Run("ping not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, m2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("userId").String().Equal(user.GetID().String())
		obj.Value("pinnedAt").String().NotEmpty()
	})
}

func TestHandlers_CreatePin(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/pin"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	m2 := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	_, err := env.MM.Pin(m2.GetID(), user.GetID())
	require.NoError(t, err)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("already pinned", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, archivedM.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("userId").String().Equal(user.GetID().String())
		obj.Value("pinnedAt").String().NotEmpty()
	})
}
