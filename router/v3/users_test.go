package v3

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/jwt"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/set"
)

func userEquals(t *testing.T, expect model.UserInfo, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.GetID().String())
	actual.Value("name").String().IsEqual(expect.GetName())
	actual.Value("displayName").String().IsEqual(expect.GetResponseDisplayName())
	actual.Value("iconFileId").String().IsEqual(expect.GetIconFileID().String())
	actual.Value("bot").Boolean().IsEqual(expect.IsBot())
	actual.Value("state").Number().IsEqual(expect.GetState())
	actual.Value("updatedAt").String().NotEmpty()
}

func TestHandlers_GetUsers(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users"
	env := Setup(t, s3)

	user := env.CreateUser(t, "xxpoxx")
	user2 := env.CreateUser(t, "sappi_red")
	deactivated := env.CreateUser(t, "deactivated")
	suspended := env.CreateUser(t, "suspended")
	err := env.Repository.UpdateUser(deactivated.GetID(), repository.UpdateUserArgs{
		UserState: optional.From(model.UserAccountStatusDeactivated),
	})
	require.NoError(t, err)
	err = env.Repository.UpdateUser(suspended.GetID(), repository.UpdateUserArgs{
		UserState: optional.From(model.UserAccountStatusSuspended),
	})
	require.NoError(t, err)

	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (include-suspended and name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("include-suspended", true).
			WithQuery("name", "xxpoxx").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (include-suspended=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("include-suspended", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(4)
	})

	t.Run("success (name=sappi_red)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("name", "sappi_red").
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		userEquals(t, user2, obj.Value(0).Object())
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

		obj.Length().IsEqual(2)
	})
}

func TestPostUserRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name     string
		Password optional.Of[string]
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{Name: "", Password: optional.From("totallySecurePassword")},
			true,
		},
		{
			"empty password",
			fields{Name: "temma", Password: optional.From("")},
			true,
		},
		{
			"too short password",
			fields{Name: "temma", Password: optional.From("password")},
			true,
		},
		{
			"success",
			fields{Name: "temma", Password: optional.From("totallySecurePassword")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostUserRequest{
				Name:     tt.fields.Name,
				Password: tt.fields.Password,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateUser(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	userSession := env.S(t, user.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostUserRequest{Name: "temma", Password: optional.From("totallySecurePassword")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (short password)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PostUserRequest{Name: "temma", Password: optional.From("password")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, userSession).
			WithJSON(&PostUserRequest{Name: "temma", Password: optional.From("totallySecurePassword")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PostUserRequest{Name: admin.GetName(), Password: optional.From("totallySecurePassword")}).
			Expect().
			Status(http.StatusConflict)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		name := random2.AlphaNumeric(20)
		obj := e.POST(path).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PostUserRequest{Name: name, Password: optional.From("totallySecurePassword")}).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("state").Number().IsEqual(model.UserAccountStatusActive)
		obj.Value("bot").Boolean().IsFalse()
		obj.Value("iconFileId").String().NotEmpty()
		obj.Value("displayName").String().IsEqual(name)
		obj.Value("name").String().IsEqual(name)
		obj.Value("twitterId").String().IsEmpty()
		obj.Value("lastOnline").IsNull()
		obj.Value("updatedAt").String().NotEmpty()
		obj.Value("tags").Array().Length().IsEqual(0)
		obj.Value("groups").Array().Length().IsEqual(0)
		obj.Value("bio").String().IsEmpty()
		obj.Value("homeChannel").IsNull()
	})
}

func TestHandlers_GetMe(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
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
			Object()

		userEquals(t, user, obj)
		obj.Value("twitterId").String().IsEmpty()
		obj.Value("lastOnline").IsNull()
		obj.Value("tags").Array().Length().IsEqual(0)
		obj.Value("groups").Array().Length().IsEqual(0)
		obj.Value("bio").String().IsEmpty()
		obj.Value("homeChannel").IsNull()
		obj.Value("permissions").Array()
	})
}

func TestHandlers_EditMe(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path).
			WithJSON(&PatchMeRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (invalid twitter id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchMeRequest{TwitterID: optional.From("ぽ")}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (invalid home channel)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchMeRequest{HomeChannel: optional.From(uuid.Must(uuid.NewV4()))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("too long display name (more than 32 letters)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PatchMeRequest{DisplayName: optional.From(strings.Repeat("a", 33))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		t.Run("success (just 32 letters)", func(t *testing.T) {
			e := env.R(t)
			e.PATCH(path).
				WithCookie(session.CookieName, s).
				WithJSON(&PatchMeRequest{DisplayName: optional.From(strings.Repeat("a", 32))}).
				Expect().
				Status(http.StatusNoContent)

			profile, err := env.Repository.GetUser(user.GetID(), true)
			require.NoError(t, err)
			assert.EqualValues(t, strings.Repeat("a", 32), profile.GetDisplayName())
		})

		t.Run("success (shorter name)", func(t *testing.T) {
			e := env.R(t)
			e.PATCH(path).
				WithCookie(session.CookieName, s).
				WithJSON(&PatchMeRequest{
					DisplayName: optional.From("po"),
					HomeChannel: optional.From(ch.ID),
				}).
				Expect().
				Status(http.StatusNoContent)

			profile, err := env.Repository.GetUser(user.GetID(), true)
			require.NoError(t, err)
			assert.EqualValues(t, "po", profile.GetDisplayName())
			if assert.True(t, profile.GetHomeChannel().Valid) {
				assert.EqualValues(t, ch.ID, profile.GetHomeChannel().V)
			}
		})
	})
}

func TestPutMyPasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Password    string
		NewPassword string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty old password",
			fields{NewPassword: "totallySecurePassword"},
			true,
		},
		{
			"empty new password",
			fields{Password: "totallySecurePassword"},
			true,
		},
		{
			"success",
			fields{Password: "totallySecurePassword", NewPassword: "evenMoreSecurePassword"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PutMyPasswordRequest{
				Password:    tt.fields.Password,
				NewPassword: tt.fields.NewPassword,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_PutMyPassword(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/password"
	env := Setup(t, common1)
	s := env.S(t, env.CreateUser(t, rand).GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("invalid body", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(echo.Map{"password": 111, "newPassword": false}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password1", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(echo.Map{"password": "test", "newPassword": "a"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password2", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(echo.Map{"password": "test", "newPassword": "アイウエオ"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("invalid password3", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(echo.Map{"password": "test", "newPassword": strings.Repeat("a", 33)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path).
			WithCookie(session.CookieName, s).
			WithJSON(echo.Map{"password": "wrong password", "newPassword": strings.Repeat("a", 20)}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		user := env.CreateUser(t, rand)

		e := env.R(t)
		newPass := strings.Repeat("a", 20)
		e.PUT(path).
			WithCookie(session.CookieName, env.S(t, user.GetID())).
			WithJSON(echo.Map{"password": "!test_test@test-", "newPassword": newPass}).
			Expect().
			Status(http.StatusNoContent)

		u, err := env.Repository.GetUser(user.GetID(), false)
		require.NoError(t, err)
		assert.NoError(t, u.Authenticate(newPass))
	})
}

func TestHandlers_GetMyQRCode(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/qr-code"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	privRaw, _ := random2.GenerateECDSAKey()
	require.NoError(t, jwt.SetupSigner(privRaw))

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (image/png)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			HasContentType("image/png")
	})

	t.Run("success (text/plain)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("token", true).
			Expect().
			Status(http.StatusOK).
			HasContentType("text/plain")
	})
}

func TestGetMyStampHistoryRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Limit int
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
			"too large limit",
			fields{Limit: 500},
			true,
		},
		{
			"success",
			fields{Limit: 50},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GetMyStampHistoryRequest{
				Limit: tt.fields.Limit,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetMyStampHistory(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/stamp-history"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	env.AddStampToMessage(t, m.GetID(), stamp.ID, user.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("limit", 500).
			Expect().
			Status(http.StatusBadRequest)
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

		first := obj.Value(0).Object()
		first.Value("stampId").String().IsEqual(stamp.ID.String())
		first.Value("datetime").String().NotEmpty()
	})
}

func TestPostMyFCMDeviceRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Token string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			true,
		},
		{
			"success",
			fields{Token: "dummy:token"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostMyFCMDeviceRequest{
				Token: tt.fields.Token,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_PostMyFCMDevice(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/fcm-device"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostMyFCMDeviceRequest{Token: "dummy:token"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (empty token)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMyFCMDeviceRequest{Token: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostMyFCMDeviceRequest{Token: "dummy:token"}).
			Expect().
			Status(http.StatusNoContent)

		tokens, err := env.Repository.GetDeviceTokens(set.UUID{user.GetID(): {}})
		require.NoError(t, err)
		if assert.Len(t, tokens, 1) {
			assert.ElementsMatch(t, tokens[user.GetID()], []string{"dummy:token"})
		}
	})
}

func TestPutUserPasswordRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		NewPassword string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty",
			fields{},
			true,
		},
		{
			"too short password",
			fields{NewPassword: "password"},
			true,
		},
		{
			"success",
			fields{NewPassword: "newPassword"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PutUserPasswordRequest{
				NewPassword: tt.fields.NewPassword,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_ChangeUserPassword(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/password"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	user2Session := env.S(t, user2.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, user1.GetID()).
			WithJSON(&PutUserPasswordRequest{NewPassword: "newPassword"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (empty password)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, user1.GetID()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PutUserPasswordRequest{NewPassword: ""}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, user1.GetID()).
			WithCookie(session.CookieName, user2Session).
			WithJSON(&PutUserPasswordRequest{NewPassword: "newPassword"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PutUserPasswordRequest{NewPassword: "newPassword"}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, user1.GetID()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PutUserPasswordRequest{NewPassword: "newPassword"}).
			Expect().
			Status(http.StatusNoContent)
	})
}

func TestHandlers_GetUser(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user.GetID()).
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
		obj := e.GET(path, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		userEquals(t, user, obj)
		obj.Value("twitterId").String().IsEmpty()
		obj.Value("lastOnline").IsNull()
		obj.Value("tags").Array().Length().IsEqual(0)
		obj.Value("groups").Array().Length().IsEqual(0)
		obj.Value("bio").String().IsEmpty()
		obj.Value("homeChannel").IsNull()
	})
}

func TestPatchUserRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		DisplayName optional.Of[string]
		TwitterID   optional.Of[string]
		Role        optional.Of[string]
		State       optional.Of[int]
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
			"too long display name",
			fields{DisplayName: optional.From(strings.Repeat("a", 33))},
			true,
		},
		{
			"success",
			fields{DisplayName: optional.From("ぽ")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchUserRequest{
				DisplayName: tt.fields.DisplayName,
				TwitterID:   tt.fields.TwitterID,
				Role:        tt.fields.Role,
				State:       tt.fields.State,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditUser(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	userSession := env.S(t, user.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID()).
			WithJSON(&PatchUserRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (too long display name)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchUserRequest{DisplayName: optional.From(strings.Repeat("a", 33))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID()).
			WithCookie(session.CookieName, userSession).
			WithJSON(&PatchUserRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchUserRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (changing user state)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user2.GetID()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchUserRequest{State: optional.From(int(model.UserAccountStatusDeactivated))}).
			Expect().
			Status(http.StatusNoContent)

		profile, err := env.Repository.GetUser(user2.GetID(), true)
		require.NoError(t, err)
		assert.EqualValues(t, model.UserAccountStatusDeactivated, profile.GetState())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, user.GetID()).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchUserRequest{DisplayName: optional.From("po")}).
			Expect().
			Status(http.StatusNoContent)

		profile, err := env.Repository.GetUser(user.GetID(), true)
		require.NoError(t, err)
		assert.EqualValues(t, "po", profile.GetDisplayName())
	})
}

func TestHandlers_GetMyChannelSubscriptions(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/subscriptions"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	err := env.CM.ChangeChannelSubscriptions(ch.ID, map[uuid.UUID]model.ChannelSubscribeLevel{
		user.GetID(): model.ChannelSubscribeLevelMarkAndNotify,
	}, false, user.GetID())
	require.NoError(t, err)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, user.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("channelId").String().IsEqual(ch.ID.String())
		first.Value("level").Number().IsEqual(model.ChannelSubscribeLevelMarkAndNotify)
	})
}

func TestPutChannelSubscribeLevelRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Level optional.Of[int]
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"invalid level",
			fields{Level: optional.From(-1)},
			true,
		},
		{
			"invalid",
			fields{},
			true,
		},
		{
			"success",
			fields{Level: optional.From(1)},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PutChannelSubscribeLevelRequest{
				Level: tt.fields.Level,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_SetChannelSubscribeLevel(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/subscriptions/{channelId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	forced := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	err := env.CM.UpdateChannel(forced.ID, repository.UpdateChannelArgs{ForcedNotification: optional.From(true)})
	require.NoError(t, err)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, ch.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (invalid level)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, ch.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PutChannelSubscribeLevelRequest{Level: optional.From(-1)}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, dm.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PutChannelSubscribeLevelRequest{Level: optional.From(2)}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (forced)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, forced.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PutChannelSubscribeLevelRequest{Level: optional.From(2)}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			WithJSON(&PutChannelSubscribeLevelRequest{Level: optional.From(2)}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PUT(path, ch.ID).
			WithCookie(session.CookieName, s).
			WithJSON(&PutChannelSubscribeLevelRequest{Level: optional.From(2)}).
			Expect().
			Status(http.StatusNoContent)

		subs, err := env.Repository.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{ChannelID: optional.From(ch.ID)})
		require.NoError(t, err)
		if assert.Len(t, subs, 1) {
			sub := subs[0]
			assert.EqualValues(t, user.GetID(), sub.UserID)
			assert.True(t, sub.Mark)
			assert.True(t, sub.Notify)
		}
	})
}

func TestHandlers_GetUserStats(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/{userId}/stats"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	stamp := env.CreateStamp(t, user.GetID(), rand)
	m := env.CreateMessage(t, user.GetID(), ch.ID, rand)
	env.AddStampToMessage(t, m.GetID(), stamp.ID, user.GetID())
	env.AddStampToMessage(t, m.GetID(), stamp.ID, user.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, user.GetID()).
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
		obj := e.GET(path, user.GetID()).
			WithCookie(session.CookieName, s).
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
		firstStamp.Value("total").Number().IsEqual(2)

		obj.Value("datetime").String().NotEmpty()
	})
}
