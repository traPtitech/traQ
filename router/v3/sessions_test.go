package v3

import (
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
)

func TestPostLoginRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name     string
		Password string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{Name: "", Password: "testTestTest"},
			true,
		},
		{
			"empty password",
			fields{Name: "po", Password: ""},
			true,
		},
		{
			"success",
			fields{Name: "po", Password: "testTestTest"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostLoginRequest{
				Name:     tt.fields.Name,
				Password: tt.fields.Password,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_Login(t *testing.T) {
	t.Parallel()

	path := "/api/v3/login"
	env := Setup(t, common1)
	user, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:        random.AlphaNumeric(20),
		DisplayName: "po",
		Role:        role.User,
		IconFileID:  uuid.Nil,
		Password:    "testTestTest",
	})
	require.NoError(t, err)
	deactivated, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:        random.AlphaNumeric(20),
		DisplayName: "po",
		Role:        role.User,
		IconFileID:  uuid.Nil,
		Password:    "testTestTest",
	})
	require.NoError(t, err)
	err = env.Repository.UpdateUser(deactivated.GetID(), repository.UpdateUserArgs{
		UserState: optional.From(model.UserAccountStatusDeactivated),
	})
	require.NoError(t, err)
	suspended, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:        random.AlphaNumeric(20),
		DisplayName: "po",
		Role:        role.User,
		IconFileID:  uuid.Nil,
		Password:    "testTestTest",
	})
	require.NoError(t, err)
	err = env.Repository.UpdateUser(suspended.GetID(), repository.UpdateUserArgs{
		UserState: optional.From(model.UserAccountStatusSuspended),
	})
	require.NoError(t, err)

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLoginRequest{Name: user.GetName()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("deactivated account", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLoginRequest{Name: deactivated.GetName(), Password: "testTestTest"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("suspended account", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLoginRequest{Name: suspended.GetName(), Password: "testTestTest"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLoginRequest{Name: random.AlphaNumeric(20), Password: "testTestTest"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLoginRequest{Name: user.GetName(), Password: "!test_test@test-"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		c := e.POST(path).
			WithJSON(&PostLoginRequest{Name: user.GetName(), Password: "testTestTest"}).
			Expect().
			Status(http.StatusNoContent).
			Cookie(session.CookieName)

		c.Name().IsEqual(session.CookieName)
		c.Value().NotEmpty()
	})

	t.Run("success with redirect", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST(path).
			WithJSON(&PostLoginRequest{Name: user.GetName(), Password: "testTestTest"}).
			WithQuery("redirect", "https://example.com").
			Expect().
			Status(http.StatusFound)

		res.Header(echo.HeaderLocation).IsEqual("https://example.com")

		c := res.Cookie(session.CookieName)
		c.Name().IsEqual(session.CookieName)
		c.Value().NotEmpty()
	})
}

func TestHandlers_Logout(t *testing.T) {
	t.Parallel()

	path := "/api/v3/logout"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())
	user2 := env.CreateUser(t, rand)
	s2 := env.S(t, user2.GetID())
	user3 := env.CreateUser(t, rand)
	s3 := env.S(t, user3.GetID())
	env.S(t, user3.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		c := e.POST(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent).
			Cookie(session.CookieName)

		c.Name().IsEqual(session.CookieName)
		c.Value().IsEmpty()
	})

	t.Run("success with redirect", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST(path).
			WithCookie(session.CookieName, s2).
			WithQuery("redirect", "https://example.com").
			Expect().
			Status(http.StatusFound)

		res.Header(echo.HeaderLocation).IsEqual("https://example.com")

		c := res.Cookie(session.CookieName)
		c.Name().IsEqual(session.CookieName)
		c.Value().IsEmpty()
	})

	t.Run("success with all session revoke", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		c := e.POST(path).
			WithCookie(session.CookieName, s3).
			WithQuery("all", true).
			Expect().
			Status(http.StatusNoContent).
			Cookie(session.CookieName)

		c.Name().IsEqual(session.CookieName)
		c.Value().IsEmpty()

		sess, err := env.SessStore.GetSessionsByUserID(user3.GetID())
		require.NoError(t, err)
		assert.Len(t, sess, 0)
	})
}

func TestHandlers_GetMySessions(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/sessions"
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
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("id").String().NotEmpty()
		first.Value("issuedAt").String().NotEmpty()
	})
}

func TestHandlers_RevokeMySession(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/sessions/{sessionId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())
	s2 := env.S(t, user.GetID())
	sess2, err := env.SessStore.GetSessionByToken(s2)
	require.NoError(t, err)
	s3 := env.S(t, user.GetID())
	sess3, err := env.SessStore.GetSessionByToken(s3)
	require.NoError(t, err)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, sess3.RefID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("non existent session", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, sess2.RefID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.SessStore.GetSessionByToken(s2)
		assert.ErrorIs(t, err, session.ErrSessionNotFound)

		// already deleted
		e.DELETE(path, sess2.RefID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err = env.SessStore.GetSessionByToken(s2)
		assert.ErrorIs(t, err, session.ErrSessionNotFound)
	})
}

func TestHandlers_GetMyTokens(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tokens"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	tok := env.IssueToken(t, client, user.GetID())
	require.ElementsMatch(t, tok.Scopes.StringArray(), []interface{}{"read"})
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

		first := obj.Value(0).Object()
		first.Value("id").String().NotEmpty()
		first.Value("clientId").String().IsEqual(client.ID)
		first.Value("scopes").Array().Length().IsEqual(1)
		first.Value("scopes").Array().Value(0).String().IsEqual("read")
		first.Value("issuedAt").String().NotEmpty()
	})
}

func TestHandlers_RevokeMyToken(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/tokens/{tokenId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	tok := env.IssueToken(t, client, user.GetID())
	tok2 := env.IssueToken(t, client, user2.GetID())
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, tok.ID).
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

	t.Run("other's token", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, tok2.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, tok.ID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetTokenByID(tok.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestHandlers_GetMyExternalAccounts(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/ex-accounts"
	env := Setup(t, common1)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		user, err := env.Repository.CreateUser(repository.CreateUserArgs{
			Name:        random.AlphaNumeric(20),
			DisplayName: "po",
			Role:        role.User,
			IconFileID:  uuid.Nil,
			Password:    "!test_test@test-",
		})
		require.NoError(t, err)
		err = env.Repository.LinkExternalUserAccount(user.GetID(), repository.LinkExternalUserAccountArgs{
			ProviderName: "traq",
			ExternalID:   "sappi_red",
			Extra:        map[string]interface{}{},
		})
		require.NoError(t, err)
		s := env.S(t, user.GetID())

		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("providerName").String().IsEqual("traq")
		first.Value("linkedAt").String().NotEmpty()
		first.Value("externalName").String().IsEqual("sappi_red")
	})

	t.Run("success with external name", func(t *testing.T) {
		t.Parallel()

		user, err := env.Repository.CreateUser(repository.CreateUserArgs{
			Name:        random.AlphaNumeric(20),
			DisplayName: "po",
			Role:        role.User,
			IconFileID:  uuid.Nil,
			Password:    "!test_test@test-",
		})
		require.NoError(t, err)
		err = env.Repository.LinkExternalUserAccount(user.GetID(), repository.LinkExternalUserAccountArgs{
			ProviderName: "traq",
			ExternalID:   "motoki317",
			Extra: map[string]interface{}{
				"externalName": "toki",
			},
		})
		require.NoError(t, err)
		s := env.S(t, user.GetID())

		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		first.Value("providerName").String().IsEqual("traq")
		first.Value("linkedAt").String().NotEmpty()
		first.Value("externalName").String().IsEqual("toki")
	})
}

func TestHandlers_LinkExternalAccount(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/ex-accounts/link"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostLinkExternalAccount{ProviderName: "traq"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("already linked", func(t *testing.T) {
		t.Parallel()

		user, err := env.Repository.CreateUser(repository.CreateUserArgs{
			Name:        random.AlphaNumeric(20),
			DisplayName: "po",
			Role:        role.User,
			IconFileID:  uuid.Nil,
			Password:    "!test_test@test-",
		})
		require.NoError(t, err)
		err = env.Repository.LinkExternalUserAccount(user.GetID(), repository.LinkExternalUserAccountArgs{
			ProviderName: "traq",
			ExternalID:   "takashi_trap",
			Extra:        map[string]interface{}{},
		})
		require.NoError(t, err)
		s := env.S(t, user.GetID())

		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostLinkExternalAccount{ProviderName: "traq"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not enabled", func(t *testing.T) {
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostLinkExternalAccount{ProviderName: "google"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostLinkExternalAccount{ProviderName: "traq"}).
			Expect().
			Status(http.StatusFound).
			Header(echo.HeaderLocation).
			IsEqual("/api/auth/traq?link=1")
	})
}

func TestHandlers_UnlinkExternalAccount(t *testing.T) {
	t.Parallel()

	path := "/api/v3/users/me/ex-accounts/unlink"
	env := Setup(t, common1)

	user, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:        random.AlphaNumeric(20),
		DisplayName: "po",
		Role:        role.User,
		IconFileID:  uuid.Nil,
		Password:    "!test_test@test-",
	})
	require.NoError(t, err)
	err = env.Repository.LinkExternalUserAccount(user.GetID(), repository.LinkExternalUserAccountArgs{
		ProviderName: "traq",
		ExternalID:   "xxpoxx",
		Extra:        map[string]interface{}{},
	})
	require.NoError(t, err)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(&PostUnlinkExternalAccount{ProviderName: "traq"}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("external account not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUnlinkExternalAccount{ProviderName: "google"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(&PostUnlinkExternalAccount{ProviderName: "traq"}).
			Expect().
			Status(http.StatusNoContent)

		externals, err := env.Repository.GetLinkedExternalUserAccounts(user.GetID())
		require.NoError(t, err)
		assert.Len(t, externals, 0)
	})
}
