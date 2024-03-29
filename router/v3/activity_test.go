package v3

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/message"
)

func TestGetActivityTimelineRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Limit      int
		All        bool
		PerChannel bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"no limit",
			fields{Limit: 0},
			false,
		},
		{
			"max limit",
			fields{Limit: 50},
			false,
		},
		{
			"negative limit",
			fields{Limit: -10},
			true,
		},
		{
			"exceeds max limit",
			fields{Limit: 100},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GetActivityTimelineRequest{
				Limit:      tt.fields.Limit,
				All:        tt.fields.All,
				PerChannel: tt.fields.PerChannel,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetActivityTimeline(t *testing.T) {
	t.Parallel()
	path := "/api/v3/activity/timeline"
	env := Setup(t, s1)
	user := env.CreateUser(t, rand)
	commonSession := env.S(t, user.GetID())

	ch1 := env.CreateChannel(t, rand)
	ch2 := env.CreateChannel(t, rand)
	_, _, err := env.Repository.ChangeChannelSubscription(ch1.ID, repository.ChangeChannelSubscriptionArgs{
		Subscription: map[uuid.UUID]model.ChannelSubscribeLevel{
			user.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
		},
	})
	require.NoError(t, err)

	m1 := env.CreateMessage(t, user.GetID(), ch1.ID, "m1")
	m2 := env.CreateMessage(t, user.GetID(), ch1.ID, "m2")
	m3 := env.CreateMessage(t, user.GetID(), ch2.ID, "m3")

	timelineMessageEquals := func(t *testing.T, expect message.Message, actual *httpexpect.Object) {
		t.Helper()
		actual.Value("id").String().IsEqual(expect.GetID().String())
		actual.Value("userId").String().IsEqual(expect.GetUserID().String())
		actual.Value("channelId").String().IsEqual(expect.GetChannelID().String())
		actual.Value("content").String().IsEqual(expect.GetText())
		actual.Value("createdAt").String().NotEmpty()
		actual.Value("updatedAt").String().NotEmpty()
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad limit 1", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("limit", "-1").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad limit 2", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("limit", "100").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (all=true, per_channel=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", true).
			WithQuery("per_channel", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(2)

		timelineMessageEquals(t, m3, obj.Value(0).Object())
		timelineMessageEquals(t, m2, obj.Value(1).Object())
	})

	t.Run("success (all=true, per_channel=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", true).
			WithQuery("per_channel", false).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(3)

		timelineMessageEquals(t, m3, obj.Value(0).Object())
		timelineMessageEquals(t, m2, obj.Value(1).Object())
		timelineMessageEquals(t, m1, obj.Value(2).Object())
	})

	t.Run("success (all=false, per_channel=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", false).
			WithQuery("per_channel", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		timelineMessageEquals(t, m2, obj.Value(0).Object())
	})

	t.Run("success (all=false, per_channel=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, commonSession).
			WithQuery("all", false).
			WithQuery("per_channel", false).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(2)

		timelineMessageEquals(t, m2, obj.Value(0).Object())
		timelineMessageEquals(t, m1, obj.Value(1).Object())
	})
}
