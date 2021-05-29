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
	"github.com/traPtitech/traQ/utils/optional"
)

func clipFolderEquals(t *testing.T, expect *model.ClipFolder, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().Equal(expect.ID.String())
	actual.Value("name").String().Equal(expect.Name)
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("ownerId").String().Equal(expect.OwnerID.String())
	actual.Value("description").String().Equal(expect.Description)
}

func TestPostClipFolderRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        string
		Description string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{Name: "", Description: "foo"},
			true,
		},
		{
			"too long name",
			fields{Name: strings.Repeat("a", 100), Description: "foo"},
			true,
		},
		{
			"success",
			fields{Name: "test", Description: "foo"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostClipFolderRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateClipFolderRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        optional.String
		Description optional.String
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
			"too long name",
			fields{Name: optional.StringFrom(strings.Repeat("a", 100))},
			true,
		},
		{
			"success",
			fields{Name: optional.StringFrom("test")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := UpdateClipFolderRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateClipFolder(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	userSession := env.S(t, user.GetID())

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
			WithCookie(session.CookieName, userSession).
			WithJSON(&PostClipFolderRequest{Name: strings.Repeat("a", 100), Description: "desc"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, userSession).
			WithJSON(&PostClipFolderRequest{Name: "nya", Description: "desc"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("name").String().Equal("nya")
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("ownerId").String().Equal(user.GetID().String())
		obj.Value("description").String().Equal("desc")

		id := obj.Value("id").String().Raw()

		// overlapping name
		obj = e.POST(path).
			WithCookie(session.CookieName, userSession).
			WithJSON(&PostClipFolderRequest{Name: "nya", Description: "desc"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty().NotEqual(id)
		obj.Value("name").String().Equal("nya")
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("ownerId").String().Equal(user.GetID().String())
		obj.Value("description").String().Equal("desc")
	})
}

func TestHandlers_GetClipFolders(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	env.CreateClipFolder(t, rand, rand, user2.GetID())
	user1Session := env.S(t, user1.GetID())

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
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().Equal(1)
		clipFolderEquals(t, cf1, obj.First().Object())
	})
}

func TestHandlers_GetClipFolder(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	user1Session := env.S(t, user1.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, cf1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, cf2.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		clipFolderEquals(t, cf1, obj)
	})
}

func TestHandlers_DeleteClipFolder(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	user1Session := env.S(t, user1.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf2.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetClipFolder(cf1.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestHandlers_EditClipFolder(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	user1Session := env.S(t, user1.GetID())

	req := &UpdateClipFolderRequest{
		Name:        optional.StringFrom("nya"),
		Description: optional.StringFrom("foo"),
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, cf1.ID.String()).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&UpdateClipFolderRequest{Name: optional.StringFrom(strings.Repeat("a", 100))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, cf2.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusNoContent)

		cf, err := env.Repository.GetClipFolder(cf1.ID)
		require.NoError(t, err)
		assert.EqualValues(t, "nya", cf.Name)
		assert.EqualValues(t, "foo", cf.Description)
	})
}

func TestHandlers_PostClipFolderMessage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}/messages"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	cf3 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	c := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user2.GetID(), c.ID, rand)
	user1Session := env.S(t, user1.GetID())

	_, err := env.Repository.AddClipFolderMessage(cf3.ID, m.GetID())
	require.NoError(t, err)

	req := &PostClipFolderMessageRequest{
		MessageID: m.GetID(),
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, cf1.ID.String()).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(map[string]interface{}{"messageId": "po"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (message not found)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PostClipFolderMessageRequest{MessageID: uuid.Must(uuid.NewV4())}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, cf2.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found (clip folder)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, cf3.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithJSON(req).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		messageEquals(t, m, obj.Value("message").Object())
		obj.Value("clippedAt").String().NotEmpty()
	})
}

func Test_clipFolderMessageQuery_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Limit  int
		Offset int
		Order  string
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
			"negative limit",
			fields{Limit: -1},
			true,
		},
		{
			"negative offset",
			fields{Offset: -1},
			true,
		},
		{
			"too large limit",
			fields{Limit: 500},
			true,
		},
		{
			"success",
			fields{Limit: 50, Offset: 50, Order: "asc"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &clipFolderMessageQuery{
				Limit:  tt.fields.Limit,
				Offset: tt.fields.Offset,
				Order:  tt.fields.Order,
			}
			if err := q.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetClipFolderMessages(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}/messages"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	c := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user2.GetID(), c.ID, rand)
	user1Session := env.S(t, user1.GetID())

	_, err := env.Repository.AddClipFolderMessage(cf1.ID, m.GetID())
	require.NoError(t, err)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, cf1.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			WithQuery("limit", -1).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, cf2.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4()).String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, cf1.ID.String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().Equal(1)

		first := obj.First().Object()
		messageEquals(t, m, first.Value("message").Object())
		first.Value("clippedAt").String().NotEmpty()
	})
}

func TestHandlers_DeleteClipFolderMessages(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clip-folders/{folderId}/messages/{messageId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	cf1 := env.CreateClipFolder(t, rand, rand, user1.GetID())
	cf2 := env.CreateClipFolder(t, rand, rand, user2.GetID())
	c := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user2.GetID(), c.ID, rand)
	m2 := env.CreateMessage(t, user1.GetID(), c.ID, rand)
	user1Session := env.S(t, user1.GetID())

	_, err := env.Repository.AddClipFolderMessage(cf1.ID, m.GetID())
	require.NoError(t, err)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf1.ID.String(), m.GetID().String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf2.ID.String(), m.GetID().String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4()).String(), m.GetID().String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (already deleted)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf1.ID, m2.GetID().String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, cf1.ID.String(), m.GetID().String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNoContent)

		messages, more, err := env.Repository.GetClipFolderMessages(cf1.ID, repository.ClipFolderMessageQuery{
			Limit:  50,
			Offset: 0,
			Asc:    false,
		})
		require.NoError(t, err)
		assert.False(t, more)
		assert.Len(t, messages, 0)

		// already deleted
		e.DELETE(path, cf1.ID.String(), m.GetID().String()).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNoContent)

		messages, more, err = env.Repository.GetClipFolderMessages(cf1.ID, repository.ClipFolderMessageQuery{
			Limit:  50,
			Offset: 0,
			Asc:    false,
		})
		require.NoError(t, err)
		assert.False(t, more)
		assert.Len(t, messages, 0)
	})
}
