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

		obj.Length().IsEqual(1)

		first := obj.First().Object()
		first.Value("channelId").String().IsEqual(ch.ID.String())
		first.Value("count").Number().IsEqual(1)
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

		obj.Value("userId").String().IsEqual(user.GetID().String())
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

		obj.Value("userId").String().IsEqual(user.GetID().String())
		obj.Value("pinnedAt").String().NotEmpty()
	})
}

func TestHandlers_RemovePin(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/pin"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	m2 := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	_, err := env.MM.Pin(m.GetID(), user.GetID())
	require.NoError(t, err)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	_, err = env.MM.Pin(archivedM.GetID(), user.GetID())
	require.NoError(t, err)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
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

	t.Run("pin not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, archivedM.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		m, err := env.MM.Get(m.GetID())
		require.NoError(t, err)
		assert.Nil(t, m.GetPin())
	})
}

func TestHandlers_GetMessageStamps(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/stamps"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	env.AddStampToMessage(t, m.GetID(), stamp.ID, user.GetID())
	s := env.S(t, user.GetID())

	var err error
	m, err = env.MM.Get(m.GetID())
	require.NoError(t, err)
	require.Len(t, m.GetStamps(), 1)

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

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.First().Object()
		messageStampEquals(t, m.GetStamps()[0], first)
	})
}

func TestPostMessageStampRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Count int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"zero count",
			fields{Count: 0},
			false,
		},
		{
			"negative count",
			fields{Count: -1},
			true,
		},
		{
			"too large count",
			fields{Count: 150},
			true,
		},
		{
			"success",
			fields{Count: 5},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostMessageStampRequest{
				Count: tt.fields.Count,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_AddMessageStamp(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/stamps/{stampId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	stamp := env.CreateStamp(t, user.GetID(), rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m.GetID(), stamp.ID).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()), stamp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("stamp not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m.GetID(), uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m.GetID(), stamp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMessageStampRequest{Count: 1000}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, archivedM.GetID(), stamp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, m.GetID(), stamp.ID).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]interface{}{}).
			Expect().
			Status(http.StatusNoContent)

		m, err := env.MM.Get(m.GetID())
		require.NoError(t, err)

		if assert.Len(t, m.GetStamps(), 1) {
			s := m.GetStamps()[0]
			assert.EqualValues(t, 1, s.Count)
			assert.EqualValues(t, stamp.ID, s.StampID)
			assert.EqualValues(t, m.GetID(), s.MessageID)
			assert.EqualValues(t, user.GetID(), s.UserID)
		}
	})
}

func TestHandlers_RemoveMessageStamp(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/stamps/{stampId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	archivedM := env.CreateMessage(t, user.GetID(), archived.ID, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	env.AddStampToMessage(t, m.GetID(), stamp.ID, user.GetID())
	env.AddStampToMessage(t, archivedM.GetID(), stamp.ID, user.GetID())
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m.GetID(), stamp.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4()), stamp.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("stamp not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m.GetID(), uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, archivedM.GetID(), stamp.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, m.GetID(), stamp.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		m, err := env.MM.Get(m.GetID())
		require.NoError(t, err)

		assert.Len(t, m.GetStamps(), 0)
	})
}

func TestHandlers_GetMessageClips(t *testing.T) {
	t.Parallel()

	path := "/api/v3/messages/{messageId}/clips"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	cf := env.CreateClipFolder(t, rand, rand, user.GetID())
	_, err := env.Repository.AddClipFolderMessage(cf.ID, m.GetID())
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

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, m.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.First().Object()
		first.Value("folderId").String().IsEqual(cf.ID.String())
		first.Value("clippedAt").String().NotEmpty()
	})
}

func TestHandlers_GetMessages(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/messages"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	m2 := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ch.ID).
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

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ch.ID).
			WithCookie(session.CookieName, s).
			WithQuery("limit", -1).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, ch.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(2)

		messageEquals(t, m2, obj.Element(0).Object())
		messageEquals(t, m, obj.Element(1).Object())
	})
}

func TestHandlers_PostMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/messages"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	s := env.S(t, user.GetID())

	req := &PostMessageRequest{
		Content: "Hello, traP",
		Embed:   true,
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ch.ID).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("archived", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, archived.ID).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ch.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMessageRequest{Content: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, ch.ID).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("userId").String().IsEqual(user.GetID().String())
		obj.Value("channelId").String().IsEqual(ch.ID.String())
		obj.Value("content").String().IsEqual("Hello, traP")
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("pinned").Boolean().False()
		obj.Value("stamps").Array().Length().IsEqual(0)

		id, err := uuid.FromString(obj.Value("id").String().Raw())
		if assert.NoError(t, err) {
			m, err := env.MM.Get(id)
			require.NoError(t, err)
			messageEquals(t, m, obj)
		}
	})
}

func TestHandlers_GetDirectMessages(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/messages"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	m := env.CreateMessage(t, user.GetID(), dm.ID, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user2.GetID()).
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

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithQuery("limit", -1).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (existing dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		messageEquals(t, m, obj.First().Object())
	})

	t.Run("success (creating dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user3.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(0)
	})
}

func TestHandlers_PostDirectMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/messages"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	s := env.S(t, user.GetID())

	req := &PostMessageRequest{
		Content: "Hello, traP",
		Embed:   true,
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, user2.GetID()).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMessageRequest{Content: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (existing dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("userId").String().IsEqual(user.GetID().String())
		obj.Value("channelId").String().IsEqual(dm.ID.String())
		obj.Value("content").String().IsEqual("Hello, traP")
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("pinned").Boolean().False()
		obj.Value("stamps").Array().Length().IsEqual(0)

		id, err := uuid.FromString(obj.Value("id").String().Raw())
		if assert.NoError(t, err) {
			m, err := env.MM.Get(id)
			require.NoError(t, err)
			messageEquals(t, m, obj)
		}
	})

	t.Run("success (creating dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, user3.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("userId").String().IsEqual(user.GetID().String())
		obj.Value("channelId").String().NotEmpty()
		obj.Value("content").String().IsEqual("Hello, traP")
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("pinned").Boolean().False()
		obj.Value("stamps").Array().Length().IsEqual(0)

		id, err := uuid.FromString(obj.Value("id").String().Raw())
		if assert.NoError(t, err) {
			m, err := env.MM.Get(id)
			require.NoError(t, err)
			messageEquals(t, m, obj)
		}
	})
}
