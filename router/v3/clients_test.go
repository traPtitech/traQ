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
	"github.com/traPtitech/traQ/utils/random"
)

func oAuth2ClientEquals(t *testing.T, expect *model.OAuth2Client, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.ID)
	actual.Value("name").String().IsEqual(expect.Name)
	actual.Value("description").String().IsEqual(expect.Description)
	actual.Value("developerId").String().IsEqual(expect.CreatorID.String())
	scopes := make([]interface{}, 0, len(expect.Scopes.StringArray()))
	for _, scope := range expect.Scopes.StringArray() {
		scopes = append(scopes, scope)
	}
	actual.Value("scopes").Array().ContainsOnly(scopes...)
	actual.Value("confidential").Boolean().IsEqual(expect.Confidential)
}

func TestHandlers_GetClients(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients"
	env := Setup(t, s1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	c1 := env.CreateOAuth2Client(t, rand, user.GetID())
	c2 := env.CreateOAuth2Client(t, rand, user2.GetID(), WithConfidential(true))
	commonSession := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
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
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)

		first := obj.Value(0).Object()
		oAuth2ClientEquals(t, c1, first)
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

		obj.Length().IsEqual(2)

		first := obj.Value(0).Object()
		second := obj.Value(1).Object()

		if first.Value("id").String().Raw() == c1.ID {
			oAuth2ClientEquals(t, c1, first)
			oAuth2ClientEquals(t, c2, second)
		} else {
			oAuth2ClientEquals(t, c2, first)
			oAuth2ClientEquals(t, c1, second)
		}
	})
}

func TestPostClientsRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name         string
		Description  string
		CallbackURL  string
		Scopes       model.AccessScopes
		Confidential bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"empty name",
			fields{
				Name:        "",
				Description: "desc",
				CallbackURL: "https://example.com",
				Scopes: map[model.AccessScope]struct{}{
					"read": {},
				},
			},
			true,
		},
		{
			"too long name",
			fields{
				Name:        strings.Repeat("a", 100),
				Description: "desc",
				CallbackURL: "https://example.com",
				Scopes: map[model.AccessScope]struct{}{
					"read": {},
				},
			},
			true,
		},
		{
			"empty callback",
			fields{
				Name:        "test",
				Description: "desc",
				CallbackURL: "",
				Scopes: map[model.AccessScope]struct{}{
					"read": {},
				},
			},
			true,
		},
		{
			"empty scopes",
			fields{
				Name:        "test",
				Description: "desc",
				CallbackURL: "https://example.com",
				Scopes:      map[model.AccessScope]struct{}{},
			},
			true,
		},
		{
			"success",
			fields{
				Name:        "test",
				Description: "desc",
				CallbackURL: "https://example.com",
				Scopes: map[model.AccessScope]struct{}{
					"read": {},
				},
			},
			false,
		},
		{
			"success (confidential client)",
			fields{
				Name:         "test",
				Description:  "desc",
				CallbackURL:  "https://example.com",
				Scopes:       map[model.AccessScope]struct{}{"read": {}},
				Confidential: true,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PostClientsRequest{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				CallbackURL: tt.fields.CallbackURL,
				Scopes:      tt.fields.Scopes,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_CreateClient(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	commonSession := env.S(t, user.GetID())

	req := &PostClientsRequest{
		Name:        "test",
		Description: "desc",
		CallbackURL: "https://example.com",
		Scopes:      map[model.AccessScope]struct{}{"read": {}},
	}

	// confidential client
	req2 := &PostClientsRequest{
		Name:         "test",
		Description:  "desc",
		CallbackURL:  "https://example.com",
		Scopes:       map[model.AccessScope]struct{}{"read": {}},
		Confidential: true,
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithJSON(req).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(&PostClientsRequest{Name: "test", Description: "desc", CallbackURL: "https://example.com"}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("developerId").String().IsEqual(user.GetID().String())
		obj.Value("description").String().IsEqual("desc")
		obj.Value("name").String().IsEqual("test")
		scopes := obj.Value("scopes").Array()
		scopes.Length().IsEqual(1)
		scopes.Value(0).String().IsEqual("read")
		obj.Value("callbackUrl").String().IsEqual("https://example.com")
		obj.Value("secret").String().NotEmpty()
		obj.Value("confidential").Boolean().IsFalse()

		c, err := env.Repository.GetClient(obj.Value("id").String().Raw())
		assert.NoError(t, err)
		oAuth2ClientEquals(t, c, obj)
	})

	t.Run("success (confidential client)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, commonSession).
			WithJSON(req2).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("developerId").String().IsEqual(user.GetID().String())
		obj.Value("description").String().IsEqual("desc")
		obj.Value("name").String().IsEqual("test")
		scopes := obj.Value("scopes").Array()
		scopes.Length().IsEqual(1)
		scopes.Value(0).String().IsEqual("read")
		obj.Value("callbackUrl").String().IsEqual("https://example.com")
		obj.Value("secret").String().NotEmpty()
		obj.Value("confidential").Boolean().IsTrue()

		c, err := env.Repository.GetClient(obj.Value("id").String().Raw())
		assert.NoError(t, err)
		oAuth2ClientEquals(t, c, obj)
	})
}

func TestHandlers_GetClient(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients/{clientId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	c1 := env.CreateOAuth2Client(t, rand, user1.GetID())
	c2 := env.CreateOAuth2Client(t, rand, user2.GetID())
	c3 := env.CreateOAuth2Client(t, rand, user1.GetID(), WithConfidential(true))
	user1Session := env.S(t, user1.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, c1.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, c2.ID).
			WithCookie(session.CookieName, user1Session).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, random.AlphaNumeric(36)).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (c1, detail=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, c1.ID).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		oAuth2ClientEquals(t, c1, obj)
	})

	t.Run("success (c2, detail=false)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, c2.ID).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		oAuth2ClientEquals(t, c2, obj)
	})

	t.Run("success (c1, detail=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, c1.ID).
			WithCookie(session.CookieName, user1Session).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		oAuth2ClientEquals(t, c1, obj)
		obj.Value("callbackUrl").String().NotEmpty()
		obj.Value("secret").String().NotEmpty()
	})

	t.Run("success (c1, admin, detail=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, c1.ID).
			WithCookie(session.CookieName, adminSession).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		oAuth2ClientEquals(t, c1, obj)
		obj.Value("callbackUrl").String().NotEmpty()
		obj.Value("secret").String().NotEmpty()
	})

	t.Run("success (c3, detail=true)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, c3.ID).
			WithCookie(session.CookieName, user1Session).
			WithQuery("detail", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		oAuth2ClientEquals(t, c3, obj)
		obj.Value("callbackUrl").String().NotEmpty()
		obj.Value("secret").String().NotEmpty()
		obj.Value("confidential").Boolean().IsTrue()
	})
}

func TestPatchClientRequest_Validate(t *testing.T) {
	t.Parallel()

	type fields struct {
		Name         optional.Of[string]
		Description  optional.Of[string]
		CallbackURL  optional.Of[string]
		DeveloperID  optional.Of[uuid.UUID]
		Confidential optional.Of[bool]
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
			fields{Name: optional.From(strings.Repeat("a", 100))},
			true,
		},
		{
			"empty description",
			fields{Description: optional.From("")},
			true,
		},
		{
			"empty callback url",
			fields{CallbackURL: optional.From("")},
			true,
		},
		{
			"nil developer id",
			fields{DeveloperID: optional.From(uuid.Nil)},
			true,
		},
		{
			"invalid developer id",
			fields{DeveloperID: optional.From(uuid.Nil)},
			true,
		},
		{
			"success",
			fields{Name: optional.From("po")},
			false,
		},
		{
			"success (confidential client)",
			fields{Confidential: optional.From(true)},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PatchClientRequest{
				Name:         tt.fields.Name,
				Description:  tt.fields.Description,
				CallbackURL:  tt.fields.CallbackURL,
				DeveloperID:  tt.fields.DeveloperID,
				Confidential: tt.fields.Confidential,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_EditClient(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients/{clientId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	c1 := env.CreateOAuth2Client(t, rand, user1.GetID())
	c2 := env.CreateOAuth2Client(t, rand, user2.GetID())
	c3 := env.CreateOAuth2Client(t, rand, user1.GetID(), WithConfidential(true))
	user1Session := env.S(t, user1.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c1.ID).
			WithJSON(&PatchClientRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c1.ID).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PatchClientRequest{Name: optional.From(strings.Repeat("a", 100))}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c2.ID).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PatchClientRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, random.AlphaNumeric(36)).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PatchClientRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (user1, c1)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c1.ID).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PatchClientRequest{Name: optional.From("po")}).
			Expect().
			Status(http.StatusNoContent)

		c, err := env.Repository.GetClient(c1.ID)
		require.NoError(t, err)
		assert.EqualValues(t, c.Name, "po")
	})

	t.Run("success (admin, c2)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c2.ID).
			WithCookie(session.CookieName, adminSession).
			WithJSON(&PatchClientRequest{Name: optional.From("po2")}).
			Expect().
			Status(http.StatusNoContent)

		c, err := env.Repository.GetClient(c2.ID)
		require.NoError(t, err)
		assert.EqualValues(t, c.Name, "po2")
	})

	t.Run("success (user1, c3, confidential)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.PATCH(path, c3.ID).
			WithCookie(session.CookieName, user1Session).
			WithJSON(&PatchClientRequest{Confidential: optional.From(true)}).
			Expect().
			Status(http.StatusNoContent)

		c, err := env.Repository.GetClient(c3.ID)
		require.NoError(t, err)
		assert.True(t, c.Confidential)
	})
}

func TestHandlers_DeleteClient(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients/{clientId}"
	env := Setup(t, common1)
	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	admin := env.CreateAdmin(t, rand)
	c1 := env.CreateOAuth2Client(t, rand, user1.GetID())
	c2 := env.CreateOAuth2Client(t, rand, user2.GetID())
	c3 := env.CreateOAuth2Client(t, rand, user1.GetID())
	c4 := env.CreateOAuth2Client(t, rand, user2.GetID())
	user1Session := env.S(t, user1.GetID())
	adminSession := env.S(t, admin.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, c3.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, c4.ID).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, random.AlphaNumeric(36)).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (user1, c1)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, c1.ID).
			WithCookie(session.CookieName, user1Session).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetClient(c1.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})

	t.Run("success (admin, c2)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, c2.ID).
			WithCookie(session.CookieName, adminSession).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.Repository.GetClient(c2.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestHandlers_RevokeClientTokens(t *testing.T) {
	t.Parallel()

	path := "/api/v3/clients/{clientId}/tokens"
	env := Setup(t, common1)

	user1 := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	client1 := env.CreateOAuth2Client(t, rand, user1.GetID())
	client2 := env.CreateOAuth2Client(t, rand, user1.GetID())

	tests := map[string]struct {
		user            model.UserInfo
		userTokenCounts map[uuid.UUID][]*model.OAuth2Client
		clientID        string
		statusCode      int
	}{
		"not logged in": {
			nil,
			nil,
			client1.ID,
			http.StatusUnauthorized,
		},
		"success": {
			user1,
			map[uuid.UUID][]*model.OAuth2Client{
				user1.GetID(): {client1},
			},
			client1.ID,
			http.StatusNoContent,
		},
		"success (同じクライアント, 複数トークン)": {
			user1,
			map[uuid.UUID][]*model.OAuth2Client{
				user1.GetID(): {client1, client1},
			},
			client1.ID,
			http.StatusNoContent,
		},
		"success (複数クライアント, 複数トークン)": {
			user1,
			map[uuid.UUID][]*model.OAuth2Client{
				user1.GetID(): {client1, client2},
			},
			client1.ID,
			http.StatusNoContent,
		},
		"success (複数ユーザー, 複数クライアント, 複数トークン)": {
			user1,
			map[uuid.UUID][]*model.OAuth2Client{
				user1.GetID(): {client1},
				user2.GetID(): {client1},
			},
			client1.ID,
			http.StatusNoContent,
		},
		"success (トークン無し)": {
			user1,
			nil,
			client1.ID,
			http.StatusNoContent,
		},
		"クライアントが存在しない": {
			user1,
			map[uuid.UUID][]*model.OAuth2Client{
				user1.GetID(): {client1},
			},
			uuid.Must(uuid.NewV7()).String(),
			http.StatusNotFound,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tokens := make(map[uuid.UUID][]*model.OAuth2Token, len(tt.userTokenCounts))
			for userID, clients := range tt.userTokenCounts {
				for _, c := range clients {
					tokens[userID] = append(tokens[userID], env.IssueToken(t, c, userID))
				}
			}

			e := env.R(t)

			if tt.user != nil {
				sess := env.S(t, tt.user.GetID())
				e.DELETE(path, tt.clientID).
					WithCookie(session.CookieName, sess).
					Expect().
					Status(tt.statusCode)
			} else {
				e.DELETE(path, tt.clientID).
					Expect().
					Status(tt.statusCode)
			}

			for userID, userTokens := range tokens {
				for _, userToken := range userTokens {
					_, err := env.Repository.GetTokenByID(userToken.ID)
					if userToken.ClientID == tt.clientID && userID == tt.user.GetID() && tt.statusCode == http.StatusNoContent {
						assert.ErrorIs(t, err, repository.ErrNotFound)
					} else {
						assert.NoError(t, err)
					}
				}
			}
		})
	}
}
