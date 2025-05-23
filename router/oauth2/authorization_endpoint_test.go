package oauth2

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
)

func TestResponseType_valid(t *testing.T) {
	t.Parallel()

	cases := [][5]bool{
		{false, false, false, false},
		{true, false, false, true},
		{false, true, false, true},
		{false, false, true, true},
		{true, true, false, true},
		{true, false, true, false},
		{false, true, true, false},
		{true, true, true, false},
	}
	for _, v := range cases {
		rt := responseType{
			Code:  v[0],
			Token: v[1],
			None:  v[2],
		}
		assert.Equal(t, v[3], rt.valid())
	}
}

func TestHandlers_AuthorizationEndpointHandler(t *testing.T) {
	t.Parallel()
	env := Setup(t, db2)
	defaultUser := env.CreateUser(t, rand)
	s := env.S(t, defaultUser.GetID())

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")

	client := &model.OAuth2Client{
		ID:           random.AlphaNumeric(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.Must(uuid.NewV7()),
		Secret:       random.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesRead,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	t.Run("Success (prompt=none)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		env.IssueToken(t, client, user.GetID(), false)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("prompt", "none").
			WithFormField("scope", "read").
			WithFormField("nonce", "nonce").
			WithCookie(session.CookieName, env.S(t, user.GetID())).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.NotEmpty(loc.Query().Get("code"))
		}

		a, err := env.Repository.GetAuthorize(loc.Query().Get("code"))
		if assert.NoError(err) {
			assert.Equal("nonce", a.Nonce)
		}
	})

	t.Run("Success (code)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		env.IssueToken(t, client, user.GetID(), false)
		s := env.S(t, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("scope", "read write").
			WithFormField("nonce", "nonce").
			WithCookie(session.CookieName, s).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		se, err := env.SessStore.GetSessionByToken(s)
		if assert.NoError(err) {
			v, err := se.Get(oauth2ContextSession)
			assert.NoError(err)
			assert.Equal("state", v.(authorizeRequest).State)
		}
	})

	t.Run("Success (GET)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		env.IssueToken(t, client, user.GetID(), false)
		s := env.S(t, user.GetID())
		e := env.R(t)
		res := e.GET("/oauth2/authorize").
			WithQuery("client_id", client.ID).
			WithQuery("response_type", "code").
			WithQuery("state", "state").
			WithQuery("scope", "read write").
			WithQuery("nonce", "nonce").
			WithCookie(session.CookieName, s).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		se, err := env.SessStore.GetSessionByToken(s)
		if assert.NoError(err) {
			v, err := se.Get(oauth2ContextSession)
			assert.NoError(err)
			assert.Equal("state", v.(authorizeRequest).State)
		}
	})

	t.Run("Success With PKCE", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		env.IssueToken(t, client, user.GetID(), false)
		s := env.S(t, user.GetID())
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("nonce", "nonce").
			WithFormField("scope", "read write").
			WithFormField("code_challenge_method", "S256").
			WithFormField("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM").
			WithCookie(session.CookieName, s).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		se, err := env.SessStore.GetSessionByToken(s)
		if assert.NoError(err) {
			v, err := se.Get(oauth2ContextSession)
			assert.NoError(err)
			assert.Equal("state", v.(authorizeRequest).State)
			assert.Equal("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", v.(authorizeRequest).CodeChallenge)
		}
	})

	t.Run("Bad Request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithForm(map[string]interface{}{}).
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Bad Request (no client)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", "").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Bad Request (unknown client)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", "unknown").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Bad Request (different redirect uri)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("redirect_uri", "http://example2.com").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Found (invalid pkce method)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("code_challenge_method", "S256").
			WithFormField("code_challenge", "ああああ").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("scope", "あいうえお").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	})

	t.Run("Found (no valid scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("scope", "write").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	})

	t.Run("Found (unknown response_type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "aiueo").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid response_type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code token none").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (GET, code with no session)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.GET("/oauth2/authorize").
			WithQuery("client_id", client.ID).
			WithQuery("response_type", "code").
			WithQuery("state", "state").
			WithQuery("scope", "read write").
			WithQuery("nonce", "nonce").
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("/login", loc.Path)
			redirectURL, err := url.Parse(loc.Query().Get("redirect"))
			if assert.NoError(err) {
				assert.Equal("/oauth2/authorize", redirectURL.Path)
				assert.Equal(client.ID, redirectURL.Query().Get("client_id"))
				assert.Equal("code", redirectURL.Query().Get("response_type"))
				assert.Equal("state", redirectURL.Query().Get("state"))
				assert.Equal("read write", redirectURL.Query().Get("scope"))
				assert.Equal("nonce", redirectURL.Query().Get("nonce"))
			}
		}
	})

	t.Run("Found (code with no session)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("scope", "read write").
			WithFormField("nonce", "nonce").
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("/login", loc.Path)
			redirectURL, err := url.Parse(loc.Query().Get("redirect"))
			if assert.NoError(err) {
				assert.Equal("/oauth2/authorize", redirectURL.Path)
				assert.Equal(client.ID, redirectURL.Query().Get("client_id"))
				assert.Equal("code", redirectURL.Query().Get("response_type"))
				assert.Equal("state", redirectURL.Query().Get("state"))
				assert.Equal("read write", redirectURL.Query().Get("scope"))
				assert.Equal("nonce", redirectURL.Query().Get("nonce"))
			}
		}
	})

	t.Run("Found (prompt=none with no session)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errLoginRequired, loc.Query().Get("error"))
		}
	})

	t.Run("Found (prompt=none without consent)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errConsentRequired, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid prompt)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "ああああ").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	})

	t.Run("Found (prompt=none with broader scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		_, err := env.Repository.IssueToken(client, user.GetID(), client.RedirectURI, scopesRead, 1000, false)
		require.NoError(t, err)
		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			WithFormField("scope", "read write").
			WithCookie(session.CookieName, env.S(t, user.GetID())).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errConsentRequired, loc.Query().Get("error"))
		}
	})

	t.Run("Forbidden (client without redirect uri)", func(t *testing.T) {
		t.Parallel()

		scopes := model.AccessScopes{}
		scopes.Add("read", "write")
		client := &model.OAuth2Client{
			ID:           random.AlphaNumeric(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    uuid.Must(uuid.NewV7()),
			Secret:       random.AlphaNumeric(36),
			Scopes:       scopes,
		}
		require.NoError(t, env.Repository.SaveClient(client))

		e := env.R(t)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Found (valid session but deactivated account)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := env.CreateUser(t, rand)
		err := env.Repository.UpdateUser(user.GetID(), repository.UpdateUserArgs{UserState: optional.From(model.UserAccountStatusDeactivated)})
		require.NoError(t, err)

		env.IssueToken(t, client, user.GetID(), false)
		s := env.S(t, user.GetID())
		e := env.R(t)
		res := e.GET("/oauth2/authorize").
			WithQuery("client_id", client.ID).
			WithQuery("response_type", "code").
			WithQuery("state", "state").
			WithQuery("scope", "read write").
			WithQuery("nonce", "nonce").
			WithCookie(session.CookieName, s).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errAccessDenied, loc.Query().Get("error"))
		}
	})
}

func TestHandlers_AuthorizationDecideHandler(t *testing.T) {
	t.Parallel()
	env := Setup(t, db2)
	user := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")
	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	client := &model.OAuth2Client{
		ID:           random.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       random.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesRead,
	}
	require.NoError(t, env.Repository.SaveClient(client))

	MakeDecideSession := func(t *testing.T, uid uuid.UUID, client *model.OAuth2Client) string {
		s, err := env.SessStore.IssueSession(uid, map[string]interface{}{
			oauth2ContextSession: authorizeRequest{
				ResponseType: "code",
				ClientID:     client.ID,
				RedirectURI:  client.RedirectURI,
				Scopes:       scopesReadWrite,
				ValidScopes:  scopesRead,
				State:        "state",
				Types:        responseType{Code: true},
				AccessTime:   time.Now(),
				Nonce:        "nonce",
			},
		})
		require.NoError(t, err)

		return s.Token()
	}

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.NotEmpty(loc.Query().Get("code"))
		}

		a, err := env.Repository.GetAuthorize(loc.Query().Get("code"))
		if assert.NoError(err) {
			assert.Equal("nonce", a.Nonce)
		}
	})

	t.Run("Bad Request (No form)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithForm(map[string]interface{}{}).
			WithCookie(session.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Forbidden (No oauth2ContextSession)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, s).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Bad Request (client not found)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, MakeDecideSession(t, user.GetID(), &model.OAuth2Client{ID: "aaaa"})).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Forbidden (client without redirect uri", func(t *testing.T) {
		t.Parallel()
		client := &model.OAuth2Client{
			ID:           random.AlphaNumeric(36),
			Name:         "test client",
			Confidential: true,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       random.AlphaNumeric(36),
			Scopes:       scopesRead,
		}
		require.NoError(t, env.Repository.SaveClient(client))
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
	})

	t.Run("Found (deny)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "deny").
			WithCookie(session.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errAccessDenied, loc.Query().Get("error"))
		}
	})

	t.Run("Found (unsupported response type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		s, err := env.SessStore.IssueSession(user.GetID(), map[string]interface{}{
			oauth2ContextSession: authorizeRequest{
				ResponseType: "code",
				ClientID:     client.ID,
				RedirectURI:  client.RedirectURI,
				Scopes:       scopesReadWrite,
				ValidScopes:  scopesRead,
				State:        "state",
				AccessTime:   time.Now(),
			},
		})
		require.NoError(t, err)

		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, s.Token()).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (timeout)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		s, err := env.SessStore.IssueSession(user.GetID(), map[string]interface{}{
			oauth2ContextSession: authorizeRequest{
				ResponseType: "code",
				ClientID:     client.ID,
				RedirectURI:  client.RedirectURI,
				Scopes:       scopesReadWrite,
				ValidScopes:  scopesRead,
				State:        "state",
				AccessTime:   time.Now().Add(-6 * time.Minute),
			},
		})
		require.NoError(t, err)

		e := env.R(t)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(session.CookieName, s.Token()).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").IsEqual("no-store")
		res.Header("Pragma").IsEqual("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errAccessDenied, loc.Query().Get("error"))
		}
	})
}
