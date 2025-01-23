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
	random2 "github.com/traPtitech/traQ/utils/random"
)

func userGroupEquals(t *testing.T, expect *model.UserGroup, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.ID.String())
	actual.Value("name").String().IsEqual(expect.Name)
	actual.Value("description").String().IsEqual(expect.Description)
	actual.Value("type").String().IsEqual(expect.Type)
	actual.Value("icon").String().IsEqual(expect.Icon.String())
	members := make([]interface{}, len(expect.Members))
	for i, member := range expect.Members {
		members[i] = map[string]interface{}{
			"id":   member.UserID.String(),
			"role": member.Role,
		}
	}
	actual.Value("members").Array().ContainsOnly(members...)
	actual.Value("createdAt").String().NotEmpty()
	actual.Value("updatedAt").String().NotEmpty()
	admins := make([]interface{}, len(expect.Admins))
	for i, admin := range expect.Admins {
		admins[i] = admin.UserID
	}
	actual.Value("admins").Array().ContainsOnly(admins...)
}

func TestHandlers_GetUserGroups(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups"
	env := Setup(t, s1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, "SysAd", "po", "", user.GetID())
	env.AddUserToUserGroup(t, user2.GetID(), ug.ID, "")
	ug, err := env.Repository.GetUserGroup(ug.ID)
	require.NoError(t, err)

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
		userGroupEquals(t, ug, obj.Value(0).Object())
	})
}

func TestPostUserGroupRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        string
		Description string
		Type        string
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
			"too long name",
			fields{Name: strings.Repeat("a", 50)},
			true,
		},
		{
			"invalid name 1",
			fields{Name: "@po"},
			true,
		},
		{
			"invalid name 2",
			fields{Name: ":po:"},
			true,
		},
		{
			"success",
			fields{Name: "graphics", Description: "graphics", Type: ""},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostUserGroupRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				Type:        tt.fields.Type,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_PostUserGroups(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	conflict := env.CreateUserGroup(t, rand, "", "", user.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostUserGroupRequest{Name: "testGroup123456"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupRequest{Name: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden (special group)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupRequest{Name: "22B", Type: "grade"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupRequest{Name: conflict.Name}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupRequest{Name: "testGroup123"}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("name").String().IsEqual("testGroup123")
		obj.Value("description").String().IsEmpty()
		obj.Value("type").String().IsEmpty()
		obj.Value("members").Array().Length().IsEqual(0)
		obj.Value("createdAt").String().NotEmpty()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("admins").Array().ContainsOnly(user.GetID())
	})
}

func TestHandlers_GetUserGroup(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "po", "", user.GetID())
	env.AddUserToUserGroup(t, user2.GetID(), ug.ID, "")
	ug, err := env.Repository.GetUserGroup(ug.ID)
	require.NoError(t, err)

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ug.ID).
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
		obj := e.GET(path, ug.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		userGroupEquals(t, ug, obj)
	})
}

func TestPatchUserGroupRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name        optional.Of[string]
		Description optional.Of[string]
		Type        optional.Of[string]
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
			"too long name",
			fields{Name: optional.From(strings.Repeat("a", 50))},
			true,
		},
		{
			"invalid name 1",
			fields{Name: optional.From("@po")},
			true,
		},
		{
			"invalid name 2",
			fields{Name: optional.From(":po:")},
			true,
		},
		{
			"success",
			fields{
				Name:        optional.From("graphics"),
				Description: optional.From(""),
				Type:        optional.From(""),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchUserGroupRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				Type:        tt.fields.Type,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditUserGroup(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())
	conflict := env.CreateUserGroup(t, rand, "", "", user.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID).
			WithJSON(&PatchUserGroupRequest{Name: optional.From(random2.AlphaNumeric(20))}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupRequest{Name: optional.From("")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug2.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupRequest{Name: optional.From(random2.AlphaNumeric(20))}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (special group)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupRequest{Type: optional.From("grade")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupRequest{Name: optional.From(conflict.Name)}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupRequest{Name: optional.From("testGroup456")}).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		assert.EqualValues(t, "testGroup456", ug.Name)
	})
}

func TestHandlers_DeleteUserGroup(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug2.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetUserGroup(ug.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestHandlers_GetUserGroupMembers(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/members"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	env.AddUserToUserGroup(t, user.GetID(), ug.ID, "")

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ug.ID).
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
		obj := e.GET(path, ug.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("id").String().IsEqual(user.GetID().String())
		first.Value("role").String().IsEmpty()
	})
}

func TestHandlers_AddUserGroupMember(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/members"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in - single", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithJSON(&PostUserGroupMemberRequest{ID: user.GetID()}).
			Expect().
			Status(http.StatusUnauthorized)
	})
	t.Run("not logged in - multiple", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithJSON(&[]PostUserGroupMemberRequest{{ID: user.GetID()}}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request - single", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupMemberRequest{ID: uuid.Must(uuid.NewV4())}).
			Expect().
			Status(http.StatusBadRequest)
	})
	t.Run("bad request - multiple", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&[]PostUserGroupMemberRequest{{ID: uuid.Must(uuid.NewV4())}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden - single", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug2.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupMemberRequest{ID: user.GetID()}).
			Expect().
			Status(http.StatusForbidden)
	})
	t.Run("forbidden - multiple", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug2.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&[]PostUserGroupMemberRequest{{ID: user.GetID()}}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found - single", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupMemberRequest{ID: user.GetID()}).
			Expect().
			Status(http.StatusNotFound)
	})
	t.Run("not found - multiple", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&[]PostUserGroupMemberRequest{{ID: user.GetID()}}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success - single", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupMemberRequest{ID: user.GetID()}).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		if assert.Len(t, ug.Members, 1) {
			m := ug.Members[0]
			assert.EqualValues(t, m.UserID, user.GetID())
			assert.EqualValues(t, m.Role, "")
		}
	})
	t.Run("success - multiple", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&[]PostUserGroupMemberRequest{{ID: user.GetID()}, {ID: user2.GetID()}}).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		if assert.Len(t, ug.Members, 2) {
			m := ug.Members[0]
			assert.EqualValues(t, m.UserID, user.GetID())
			assert.EqualValues(t, m.Role, "")

			m = ug.Members[1]
			assert.EqualValues(t, m.UserID, user2.GetID())
			assert.EqualValues(t, m.Role, "")
		}
	})
}

func TestPatchUserGroupMemberRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Role string
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
			"too long role name",
			fields{Role: strings.Repeat("a", 150)},
			true,
		},
		{
			"success",
			fields{Role: "po"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchUserGroupMemberRequest{
				Role: tt.fields.Role,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditUserGroupMember(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/members/{userId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())
	env.AddUserToUserGroup(t, user.GetID(), ug.ID, "")
	env.AddUserToUserGroup(t, user.GetID(), ug2.ID, "")

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID, user.GetID()).
			WithJSON(&PatchUserGroupMemberRequest{Role: "po"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (non existent member)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID, user2.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupMemberRequest{Role: "po"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (role name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupMemberRequest{Role: strings.Repeat("a", 150)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug2.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupMemberRequest{Role: "po"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("user group not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4()), user.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupMemberRequest{Role: "po"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, ug.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchUserGroupMemberRequest{Role: "po"}).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		if assert.Len(t, ug.Members, 1) {
			m := ug.Members[0]
			assert.EqualValues(t, m.UserID, user.GetID())
			assert.EqualValues(t, m.Role, "po")
		}
	})
}

func TestHandlers_RemoveUserGroupMember(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/members/{userId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())
	env.AddUserToUserGroup(t, user.GetID(), ug.ID, "")
	env.AddUserToUserGroup(t, user.GetID(), ug2.ID, "")

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID, user.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug2.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("user group not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4()), user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("already removed", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID, user2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		assert.Len(t, ug.Members, 0)

		// already removed
		e.DELETE(path, ug.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_GetUserGroupAdmins(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/admins"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ug.ID).
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

	t.Run("success (my group)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ug.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(user.GetID())
	})

	t.Run("success (other group)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, ug2.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array().
			ContainsOnly(user2.GetID())
	})
}

func TestHandlers_AddUserGroupAdmin(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/admins"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	wh := env.CreateWebhook(t, rand, user.GetID(), ch.ID)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user2.GetID())

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithJSON(&PostUserGroupAdminRequest{ID: user2.GetID()}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (invalid user id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupAdminRequest{ID: wh.GetID()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug2.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupAdminRequest{ID: user.GetID()}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupAdminRequest{ID: user2.GetID()}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path, ug.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUserGroupAdminRequest{ID: user2.GetID()}).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)

		admins := make([]uuid.UUID, len(ug.Admins))
		for i, admin := range ug.Admins {
			admins[i] = admin.UserID
		}
		assert.ElementsMatch(t, admins, []uuid.UUID{user.GetID(), user2.GetID()})
	})
}

func TestHandlers_RemoveUserGroupAdmin(t *testing.T) {
	t.Parallel()

	path := "/api/v3/groups/{groupId}/admins/{userId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)

	ug := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug2 := env.CreateUserGroup(t, rand, "", "", user.GetID())
	ug3 := env.CreateUserGroup(t, rand, "", "", user2.GetID())
	require.NoError(t, env.Repository.AddUserToGroupAdmin(user3.GetID(), ug.ID))
	require.NoError(t, env.Repository.AddUserToGroupAdmin(user3.GetID(), ug3.ID))

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID, user3.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (last admin)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug2.ID, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug3.ID, user3.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, ug.ID, user3.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		ug, err := env.Repository.GetUserGroup(ug.ID)
		require.NoError(t, err)
		if assert.Len(t, ug.Admins, 1) {
			admin := ug.Admins[0]
			assert.EqualValues(t, user.GetID(), admin.UserID)
		}
	})
}
