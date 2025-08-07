package v3

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/set"
)

func messageStampEquals(t *testing.T, expect model.MessageStamp, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("userId").String().IsEqual(expect.UserID.String())
	actual.Value("stampId").String().IsEqual(expect.StampID.String())
	actual.Value("count").Number().IsEqual(expect.Count)
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
}

func messageEquals(t *testing.T, expect message.Message, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.GetID().String())
	actual.Value("userId").String().IsEqual(expect.GetUserID().String())
	actual.Value("channelId").String().IsEqual(expect.GetChannelID().String())
	actual.Value("content").String().IsEqual(expect.GetText())
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
	actual.Value("pinned").Boolean().IsEqual(expect.GetPin() != nil)

	stamps := actual.Value("stamps").Array()
	stamps.Length().IsEqual(len(expect.GetStamps()))
	for i, ex := range expect.GetStamps() {
		messageStampEquals(t, ex, stamps.Value(i).Object())
	}
}

func channelEquals(t *testing.T, expect *model.Channel, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.ID.String())
	if expect.ParentID == uuid.Nil {
		actual.Value("parentId").IsNull()
	} else {
		actual.Value("parentId").String().IsEqual(expect.ParentID.String())
	}
	actual.Value("archived").Boolean().IsEqual(expect.IsArchived())
	actual.Value("force").Boolean().IsEqual(expect.IsForced)
	actual.Value("topic").String().IsEqual(expect.Topic)
	actual.Value("name").String().IsEqual(expect.Name)
	childIDs := make([]interface{}, 0, len(expect.ChildrenID))
	for _, childID := range expect.ChildrenID {
		childIDs = append(childIDs, childID)
	}
	actual.Value("children").Array().ContainsOnly(childIDs...)
}

func channelListElementEquals(t *testing.T, expect []*model.Channel, actual *httpexpect.Array) {
	t.Helper()
	// copy to avoid modifying `expect`
	expectCopy := make([]*model.Channel, len(expect))
	copy(expectCopy, expect)

	channelCount := int(actual.Length().IsEqual(len(expect)).Raw())
	// do not use `expect`, use `expectCopy` instead
	for i := range channelCount {
		channelObj := actual.Value(i).Object()
		channelIDString := channelObj.Value("id").String().Raw()

		j := slices.IndexFunc(expectCopy, func(c *model.Channel) bool {
			return c.ID.String() == channelIDString
		})
		assert.NotEqual(t, -1, j, "channel not found in expect list")
		expectObj := expectCopy[j]
		channelEquals(t, expectObj, channelObj)
		expectCopy = append(expectCopy[:j], expectCopy[j+1:]...) // remove found channel from `expect`
	}
	assert.Empty(t, expectCopy)
}

func TestHandlers_GetChannels(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels"
	env := Setup(t, s2)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	subchannel := env.CreateSubchannel(t, channel, rand)
	dm := env.CreateDMChannel(t, user1.GetID(), user2.GetID())
	user1Session := env.S(t, user1.GetID())
	user2Session := env.S(t, user2.GetID())
	user3Session := env.S(t, user3.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (include-dms=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		channelListElementEquals(t, []*model.Channel{channel, subchannel}, public)
	})

	t.Run("success (include-dm=true, user1)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user1Session).
			WithQuery("include-dm", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		channelListElementEquals(t, []*model.Channel{channel, subchannel}, public)

		dms := obj.Value("dm").Array()
		dms.Length().IsEqual(1)
		firstDM := dms.Value(0).Object()
		firstDM.Value("id").String().IsEqual(dm.ID.String())
		firstDM.Value("userId").String().IsEqual(user2.GetID().String())
	})

	t.Run("success (include-dm=true, user2)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user2Session).
			WithQuery("include-dm", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		channelListElementEquals(t, []*model.Channel{channel, subchannel}, public)

		dms := obj.Value("dm").Array()
		dms.Length().IsEqual(1)
		firstDM := dms.Value(0).Object()
		firstDM.Value("id").String().IsEqual(dm.ID.String())
		firstDM.Value("userId").String().IsEqual(user1.GetID().String())
	})

	t.Run("success (include-dm=true, user3)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user3Session).
			WithQuery("include-dm", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		channelListElementEquals(t, []*model.Channel{channel, subchannel}, public)

		dms := obj.Value("dm").Array()
		dms.Length().IsEqual(0)
	})

	t.Run("success (path=*, user1)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user1Session).
			WithQuery("path", env.CM.GetChannelPathFromID(channel.ID)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		public.Length().IsEqual(1)
		channelEquals(t, channel, public.Value(0).Object())
	})

	t.Run("success (path=*/*, user1)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, user1Session).
			WithQuery("path", env.CM.GetChannelPathFromID(subchannel.ID)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		public := obj.Value("public").Array()
		public.Length().IsEqual(1)
		channelEquals(t, subchannel, public.Value(0).Object())
	})

	t.Run("invalid path", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		_ = e.GET(path).
			WithCookie(session.CookieName, user1Session).
			WithQuery("path", "invalid-channel-path").
			Expect().
			Status(http.StatusNotFound)
	})
}

func TestPostChannelRequest_Validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		Name   string
		Parent optional.Of[uuid.UUID]
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{},
			true,
		},
		{
			"invalid name",
			fields{Name: "チャンネル"},
			true,
		},
		{
			"too long name",
			fields{Name: strings.Repeat("a", 50)},
			true,
		},
		{
			"success",
			fields{Name: "po"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostChannelRequest{
				Name:   tt.fields.Name,
				Parent: tt.fields.Parent,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateChannels(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostChannelRequest{Name: "po"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (invalid name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostChannelRequest{Name: "チャンネル"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		cname1 := random.AlphaNumeric(20)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostChannelRequest{Name: cname1}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("parentId").IsNull()
		obj.Value("archived").Boolean().IsFalse()
		obj.Value("force").Boolean().IsFalse()
		obj.Value("topic").String().IsEmpty()
		obj.Value("name").String().IsEqual(cname1)
		obj.Value("children").Array().Length().IsEqual(0)

		c1, err := uuid.FromString(obj.Value("id").String().Raw())
		require.NoError(t, err)

		cname2 := random.AlphaNumeric(20)
		obj = e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostChannelRequest{Name: cname2, Parent: optional.From(c1)}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("parentId").String().IsEqual(c1.String())
		obj.Value("archived").Boolean().IsFalse()
		obj.Value("force").Boolean().IsFalse()
		obj.Value("topic").String().IsEmpty()
		obj.Value("name").String().IsEqual(cname2)
		obj.Value("children").Array().Length().IsEqual(0)

		ch, err := env.CM.GetChannel(c1)
		require.NoError(t, err)
		if assert.Len(t, ch.ChildrenID, 1) {
			assert.EqualValues(t, ch.ChildrenID[0].String(), obj.Value("id").String().Raw())
		}
	})
}

func TestHandlers_GetChannel(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels/{channelId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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
		obj := e.GET(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		channelEquals(t, channel, obj)
	})
}

func TestPatchChannelRequest_Validate(t *testing.T) {
	t.Parallel()
	type fields struct {
		Name     optional.Of[string]
		Archived optional.Of[bool]
		Force    optional.Of[bool]
		Parent   optional.Of[uuid.UUID]
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			false,
		},
		{
			"empty name",
			fields{Name: optional.From("")},
			true,
		},
		{
			"invalid name",
			fields{Name: optional.From("チャンネル")},
			true,
		},
		{
			"too long name",
			fields{Name: optional.From(strings.Repeat("a", 50))},
			true,
		},
		{
			"success",
			fields{
				Name:     optional.From("po"),
				Archived: optional.From(true),
				Force:    optional.From(true),
				Parent:   optional.From(uuid.Must(uuid.NewV4())),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchChannelRequest{
				Name:     tt.fields.Name,
				Archived: tt.fields.Archived,
				Force:    tt.fields.Force,
				Parent:   tt.fields.Parent,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditChannel(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels/{channelId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	userSession := env.S(t, user.GetID())
	adminSession := env.S(t, admin.GetID())

	channel := env.CreateChannel(t, rand)
	parent := env.CreateChannel(t, rand)
	unarchived := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, admin.GetID()))

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, channel.ID).
			WithJSON(&PatchChannelRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, channel.ID).
			WithCookie(session.CookieName, userSession).
			WithJSON(&PatchChannelRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("bad request (invalid name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, channel.ID).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchChannelRequest{Name: optional.From("チャンネル")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchChannelRequest{Name: optional.From("チャンネル")}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (archive)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, unarchived.ID).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchChannelRequest{Archived: optional.From(true)}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.CM.GetChannel(unarchived.ID)
		require.NoError(t, err)
		assert.True(t, ch.IsArchived())
	})

	t.Run("success (unarchive)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, archived.ID).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchChannelRequest{Archived: optional.From(false)}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.CM.GetChannel(archived.ID)
		require.NoError(t, err)
		assert.False(t, ch.IsArchived())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		newName := random.AlphaNumeric(20)
		e.PATCH(path, channel.ID).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchChannelRequest{Name: optional.From(newName), Force: optional.From(true), Parent: optional.From(parent.ID)}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.CM.GetChannel(channel.ID)
		require.NoError(t, err)
		assert.True(t, ch.IsForced)
		assert.EqualValues(t, newName, ch.Name)
		assert.EqualValues(t, parent.ID, ch.ParentID)
	})
}

func TestHandlers_GetChannelStats(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels/{channelId}/stats"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	m1 := env.CreateMessage(t, user.GetID(), channel.ID, rand)
	m2 := env.CreateMessage(t, user.GetID(), channel.ID, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	env.AddStampToMessage(t, m1.GetID(), stamp.ID, user.GetID())
	env.AddStampToMessage(t, m2.GetID(), stamp.ID, user.GetID())
	env.DeleteMessage(t, m2.GetID())
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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

	t.Run("success (exclude-deleted-messages=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, channel.ID).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("totalMessageCount").Number().IsEqual(2)

		stamps := obj.Value("stamps").Array()
		stamps.Length().IsEqual(1)
		firstStamp := stamps.Value(0).Object()
		firstStamp.Value("id").String().IsEqual(stamp.ID.String())
		firstStamp.Value("count").Number().IsEqual(2)
		firstStamp.Value("total").Number().IsEqual(2)

		users := obj.Value("users").Array()
		users.Length().IsEqual(1)
		firstUser := users.Value(0).Object()
		firstUser.Value("id").String().IsEqual(user.GetID().String())
		firstUser.Value("messageCount").Number().IsEqual(2)
	})

	t.Run("success (exclude-deleted-messages=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, channel.ID).
			WithCookie(session.CookieName, commonSession).
			WithQuery("exclude-deleted-messages", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("totalMessageCount").Number().IsEqual(1)

		stamps := obj.Value("stamps").Array()
		stamps.Length().IsEqual(1)
		firstStamp := stamps.Value(0).Object()
		firstStamp.Value("id").String().IsEqual(stamp.ID.String())
		firstStamp.Value("count").Number().IsEqual(1)
		firstStamp.Value("total").Number().IsEqual(1)

		users := obj.Value("users").Array()
		users.Length().IsEqual(1)
		firstUser := users.Value(0).Object()
		firstUser.Value("id").String().IsEqual(user.GetID().String())
		firstUser.Value("messageCount").Number().IsEqual(1)
	})
}

func TestHandlers_GetChannelTopic(t *testing.T) {
	t.Parallel()
	path := "/api/v3/channels/{channelId}/topic"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.UpdateChannel(channel.ID, repository.UpdateChannelArgs{Topic: optional.From("this is channel topic")}))
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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
		e.GET(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("topic").
			IsEqual("this is channel topic")
	})
}

func TestHandlers_EditChannelTopic(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/topic"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	archived := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, channel.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: "test"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("bad request (archived)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, archived.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: "test"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (over 500 letters)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: strings.Repeat("a", 501)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: "test"}).
			Expect().
			Status(http.StatusNoContent)

		ch, err := env.CM.GetChannel(channel.ID)
		require.NoError(t, err)
		assert.EqualValues(t, "test", ch.Topic)

		e.PUT(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: ""}).
			Expect().
			Status(http.StatusNoContent)

		ch, err = env.CM.GetChannel(channel.ID)
		require.NoError(t, err)
		assert.EqualValues(t, "", ch.Topic)

		e.PUT(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelTopicRequest{Topic: strings.Repeat("a", 500)}).
			Expect().
			Status(http.StatusNoContent)

		ch, err = env.CM.GetChannel(channel.ID)
		require.NoError(t, err)
		assert.EqualValues(t, strings.Repeat("a", 500), ch.Topic)
	})
}

func TestHandlers_GetChannelPins(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/pins"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	m1 := env.CreateMessage(t, user.GetID(), channel.ID, rand)
	env.CreateMessage(t, user.GetID(), channel.ID, rand)
	_, err := env.MM.Pin(m1.GetID(), user.GetID())
	require.NoError(t, err)
	m1, err = env.MM.Get(m1.GetID())
	require.NoError(t, err)
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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
		obj := e.GET(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("userId").String().IsEqual(user.GetID().String())
		first.Value("pinnedAt").String().NotEmpty()

		messageEquals(t, m1, first.Value("message").Object())
	})
}

func Test_channelEventsQuery_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Limit     int
		Offset    int
		Since     optional.Of[time.Time]
		Until     optional.Of[time.Time]
		Inclusive bool
		Order     string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"zero",
			fields{},
			false,
		},
		{
			"negative offset",
			fields{Offset: -1},
			true,
		},
		{
			"too large limit",
			fields{Limit: 1000},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &channelEventsQuery{
				Limit:     tt.fields.Limit,
				Offset:    tt.fields.Offset,
				Since:     tt.fields.Since,
				Until:     tt.fields.Until,
				Inclusive: tt.fields.Inclusive,
				Order:     tt.fields.Order,
			}
			if err := q.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetChannelEvents(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/events"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	// TopicChanged
	require.NoError(t, env.CM.UpdateChannel(channel.ID, repository.UpdateChannelArgs{UpdaterID: user.GetID(), Topic: optional.From("test topic")}))
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithQuery("limit", -1).
			Expect().
			Status(http.StatusBadRequest)
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
		first.Value("type").String().IsEqual("TopicChanged")
		first.Value("datetime").String().NotEmpty()

		detail := first.Value("detail").Object()
		detail.Value("userId").String().IsEqual(user.GetID().String())
		detail.Value("before").String().IsEqual("")
		detail.Value("after").String().IsEqual("test topic")
	})
}

func TestHandlers_GetChannelSubscribers(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/subscribers"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	err := env.CM.ChangeChannelSubscriptions(channel.ID, map[uuid.UUID]model.ChannelSubscribeLevel{
		user.GetID():  model.ChannelSubscribeLevelMarkAndNotify,
		user2.GetID(): model.ChannelSubscribeLevelNone,
		user3.GetID(): model.ChannelSubscribeLevelMark,
	}, false, user.GetID())
	require.NoError(t, err)
	forced := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.UpdateChannel(forced.ID, repository.UpdateChannelArgs{ForcedNotification: optional.From(true)}))
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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

	t.Run("forbidden (forced)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, forced.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, dm.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
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
		obj.Value(0).String().IsEqual(user.GetID().String())
	})
}

func TestHandlers_GetChannelAudiences(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/audiences"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	channel := env.CreateChannel(t, rand)
	err := env.CM.ChangeChannelSubscriptions(channel.ID, map[uuid.UUID]model.ChannelSubscribeLevel{
		user.GetID():  model.ChannelSubscribeLevelMark,
		user2.GetID(): model.ChannelSubscribeLevelNone,
		user3.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
	}, false, user.GetID())
	require.NoError(t, err)
	forced := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.UpdateChannel(forced.ID, repository.UpdateChannelArgs{ForcedNotification: optional.From(true)}))
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, channel.ID).
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

	t.Run("forbidden (forced)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, forced.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, dm.ID.String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusForbidden)
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
		obj.Value(0).String().IsEqual(user.GetID().String())
	})
}

func TestHandlers_SetChannelSubscribers(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/subscribers"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	// user: none
	// user2: mark and notify
	channel := env.CreateChannel(t, rand)
	err := env.CM.ChangeChannelSubscriptions(channel.ID, map[uuid.UUID]model.ChannelSubscribeLevel{
		user2.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
	}, false, user.GetID())
	require.NoError(t, err)
	forced := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.UpdateChannel(forced.ID, repository.UpdateChannelArgs{ForcedNotification: optional.From(true)}))
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, channel.ID).
			WithJSON(&PutChannelSubscribersRequest{On: set.UUID{user.GetID(): struct{}{}}}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelSubscribersRequest{On: set.UUID{user.GetID(): struct{}{}}}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("forbidden (forced)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, forced.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelSubscribersRequest{On: set.UUID{user.GetID(): struct{}{}}}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, dm.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelSubscribersRequest{On: set.UUID{user.GetID(): struct{}{}}}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PutChannelSubscribersRequest{On: set.UUID{user.GetID(): struct{}{}}}).
			Expect().
			Status(http.StatusNoContent)

		subs, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{ChannelID: optional.From(channel.ID)})
		require.NoError(t, err)

		if assert.Len(t, subs, 1) {
			require.EqualValues(t, channel.ID, subs[0].ChannelID)
			assert.EqualValues(t, user.GetID(), subs[0].UserID)
			assert.True(t, subs[0].Mark)
			assert.True(t, subs[0].Notify)
		}
	})
}

func TestHandlers_EditChannelSubscribers(t *testing.T) {
	t.Parallel()

	path := "/api/v3/channels/{channelId}/subscribers"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	user4 := env.CreateUser(t, rand)
	user5 := env.CreateUser(t, rand)
	// user: none
	// user2: mark
	// user3: mark and notify
	// user4: mark and notify
	// user5: mark and notify
	channel := env.CreateChannel(t, rand)
	err := env.CM.ChangeChannelSubscriptions(channel.ID, map[uuid.UUID]model.ChannelSubscribeLevel{
		user2.GetID(): model.ChannelSubscribeLevelMark,
		user3.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
		user4.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
		user5.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
	}, false, user.GetID())
	require.NoError(t, err)
	forced := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.UpdateChannel(forced.ID, repository.UpdateChannelArgs{ForcedNotification: optional.From(true)}))
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	commonSession := env.S(t, user.GetID())

	req := &PatchChannelSubscribersRequest{
		On: set.UUID{
			user.GetID():  struct{}{},
			user5.GetID(): struct{}{},
		},
		Off: set.UUID{
			user2.GetID(): struct{}{},
			user3.GetID(): struct{}{},
			user5.GetID(): struct{}{},
		},
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, channel.ID).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("forbidden (forced)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, forced.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, dm.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, channel.ID.String()).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req).
			Expect().
			Status(http.StatusNoContent)

		subs, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{ChannelID: optional.From(channel.ID)})
		require.NoError(t, err)

		if assert.Len(t, subs, 4) {
			require.EqualValues(t, channel.ID, subs[0].ChannelID)
			require.EqualValues(t, channel.ID, subs[1].ChannelID)
			require.EqualValues(t, channel.ID, subs[2].ChannelID)
			require.EqualValues(t, channel.ID, subs[3].ChannelID)

			subsMap := map[uuid.UUID]model.ChannelSubscribeLevel{}
			for _, subscription := range subs {
				if subscription.Mark && subscription.Notify {
					subsMap[subscription.UserID] = model.ChannelSubscribeLevelMarkAndNotify
				} else if subscription.Mark {
					subsMap[subscription.UserID] = model.ChannelSubscribeLevelMark
				} else {
					subsMap[subscription.UserID] = model.ChannelSubscribeLevelNone
				}
			}

			assert.EqualValues(t, model.ChannelSubscribeLevelMarkAndNotify, subsMap[user.GetID()])
			assert.EqualValues(t, model.ChannelSubscribeLevelMark, subsMap[user2.GetID()])
			assert.EqualValues(t, model.ChannelSubscribeLevelMarkAndNotify, subsMap[user4.GetID()])
			assert.EqualValues(t, model.ChannelSubscribeLevelMarkAndNotify, subsMap[user5.GetID()])
		}
	})
}

func TestHandlers_GetUserDMChannel(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/dm-channel"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user2.GetID().String()).
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

	t.Run("success (existing dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user2.GetID().String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().IsEqual(dm.ID.String())
		obj.Value("userId").String().IsEqual(user2.GetID().String())
	})

	t.Run("success (creating dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user3.GetID().String()).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty().NotEqual(dm.ID.String())
		obj.Value("userId").String().IsEqual(user3.GetID().String())
	})
}
