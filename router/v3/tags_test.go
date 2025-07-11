package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
)

func userTagEquals(t *testing.T, expect model.UserTag, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("tagId").String().IsEqual(expect.GetTagID().String())
	actual.Value("tag").String().IsEqual(expect.GetTag())
	actual.Value("isLocked").Boolean().IsEqual(expect.GetIsLocked())
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
}

func TestHandlers_GetUserTags(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/tags"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user2.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV7())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		userTagEquals(t, ut, obj.Value(0).Object())
	})
}

func TestHandlers_GetMyUserTags(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tags"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user.GetID())

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
		userTagEquals(t, ut, obj.Value(0).Object())
	})
}

func TestPostUserTagRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Tag string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty tag name",
			fields{},
			true,
		},
		{
			"too long tag name",
			fields{Tag: strings.Repeat("a", 150)},
			true,
		},
		{
			"success",
			fields{Tag: "curiousなきゅうり🥒 curiousなきゅうり🥒+++"}, // 30 runes
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostUserTagRequest{
				Tag: tt.fields.Tag,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_AddUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/tags"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)

	env.AddTag(t, "409_conflict", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, user2.GetID()).
			WithJSON(&PostUserTagRequest{Tag: "†俺があやせだ†"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, wh.GetBotUserID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "†俺があやせだ†"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "†俺があやせだ†"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV7())).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "†俺があやせだ†"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "409_conflict"}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "†俺があやせだ†"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("tagId").String().NotEmpty()
		obj.Value("tag").String().IsEqual("†俺があやせだ†")
		obj.Value("isLocked").Boolean().IsFalse()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
	})
}

func TestHandlers_AddMyUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tags"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	env.AddTag(t, "409_conflict", user.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostUserTagRequest{Tag: "普段からJSとか触ってます"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "409_conflict"}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserTagRequest{Tag: "普段からJSとか触ってます"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("tagId").String().NotEmpty()
		obj.Value("tag").String().IsEqual("普段からJSとか触ってます")
		obj.Value("isLocked").Boolean().IsFalse()
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
	})
}

func TestHandlers_EditUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/tags/{tagId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user.GetID())
	ut2 := env.AddTag(t, "test", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID(), ut.GetTagID()).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID(), ut.GetTagID()).
			WithJSON(map[string]interface{}{"isLocked": "po"}).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user2.GetID(), ut2.GetTagID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("user not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()), ut.GetTagID()).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("tag not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID(), uuid.Must(uuid.NewV4())).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID(), ut.GetTagID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			Expect().
			Status(http.StatusNoContent)

		ut, err := env.Repository.GetUserTag(user.GetID(), ut.GetTagID())
		require.NoError(t, err)
		assert.True(t, ut.GetIsLocked())
	})
}

func TestHandlers_EditMyUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tags/{tagId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ut.GetTagID()).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ut.GetTagID()).
			WithJSON(map[string]interface{}{"isLocked": "po"}).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("tag not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4())).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ut.GetTagID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserTagRequest{IsLocked: true}).
			Expect().
			Status(http.StatusNoContent)

		ut, err := env.Repository.GetUserTag(user.GetID(), ut.GetTagID())
		require.NoError(t, err)
		assert.True(t, ut.GetIsLocked())
	})
}

func TestHandlers_RemoveUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/tags/{tagId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user2.GetID())
	ut2 := env.AddTag(t, "test2", user2.GetID())
	locked := env.AddTag(t, "test3", user2.GetID())
	require.NoError(t, env.Repository.ChangeUserTagLock(user2.GetID(), locked.GetTagID(), true))

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, user2.GetID(), ut2.GetTagID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, user2.GetID(), locked.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, user2.GetID(), ut.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetUserTag(user2.GetID(), ut.GetTagID())
		assert.ErrorIs(t, err, repository.ErrNotFound)

		// already removed
		e.DELETE(path, user2.GetID(), ut.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_RemoveMyUserTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tags/{tagId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	ut := env.AddTag(t, "test", user.GetID())
	ut2 := env.AddTag(t, "test2", user.GetID())
	locked := env.AddTag(t, "test3", user.GetID())
	require.NoError(t, env.Repository.ChangeUserTagLock(user.GetID(), locked.GetTagID(), true))

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ut2.GetTagID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, locked.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ut.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetUserTag(user.GetID(), ut.GetTagID())
		assert.ErrorIs(t, err, repository.ErrNotFound)

		// already removed
		e.DELETE(path, ut.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_GetTag(t *testing.T) {
	t.Parallel()

	path := "/api/v3/tags/{tagId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ut := env.AddTag(t, rand, user.GetID())
	env.AddTag(t, ut.GetTag(), user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ut.GetTagID()).
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
		obj := e.GET(path, ut.GetTagID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().IsEqual(ut.GetTagID().String())
		obj.Value("tag").String().IsEqual(ut.GetTag())
		obj.Value("users").Array().ContainsOnly(user.GetID(), user2.GetID())
	})
}
