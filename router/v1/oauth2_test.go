package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
	repo, server, _, _, session, _ := setup(t, common7)

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")

	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesRead,
	}
	require.NoError(t, repo.SaveClient(client))

	t.Run("Success (prompt=none)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := mustMakeUser(t, repo, random)
		mustIssueToken(t, repo, client, user.ID, false)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("prompt", "none").
			WithFormField("scope", "read").
			WithFormField("nonce", "nonce").
			WithCookie(sessions.CookieName, generateSession(t, user.ID)).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.NotEmpty(loc.Query().Get("code"))
		}

		a, err := repo.GetAuthorize(loc.Query().Get("code"))
		if assert.NoError(err) {
			assert.Equal("nonce", a.Nonce)
		}
	})

	t.Run("Success (code)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("scope", "read write").
			WithFormField("nonce", "nonce").
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		s, err := sessions.GetByToken(res.Cookie(sessions.CookieName).Value().Raw())
		if assert.NoError(err) {
			assert.Equal("state", s.Get(oauth2ContextSession).(authorizeRequest).State)
		}
	})

	t.Run("Success (GET)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.GET("/api/1.0/oauth2/authorize").
			WithQuery("client_id", client.ID).
			WithQuery("response_type", "code").
			WithQuery("state", "state").
			WithQuery("scope", "read write").
			WithQuery("nonce", "nonce").
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		s, err := sessions.GetByToken(res.Cookie(sessions.CookieName).Value().Raw())
		if assert.NoError(err) {
			assert.Equal("state", s.Get(oauth2ContextSession).(authorizeRequest).State)
		}
	})

	t.Run("Success With PKCE", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("nonce", "nonce").
			WithFormField("scope", "read write").
			WithFormField("code_challenge_method", "S256").
			WithFormField("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM").
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.Equal(client.ID, loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		s, err := sessions.GetByToken(res.Cookie(sessions.CookieName).Value().Raw())
		if assert.NoError(err) {
			assert.Equal("state", s.Get(oauth2ContextSession).(authorizeRequest).State)
			assert.Equal("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", s.Get(oauth2ContextSession).(authorizeRequest).CodeChallenge)
		}
	})

	t.Run("Bad Request", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (no client)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", "").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (unknown client)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", "unknown").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (different redirect uri)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("redirect_uri", "http://example2.com").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Found (invalid pkce method)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("code_challenge_method", "S256").
			WithFormField("code_challenge", "ああああ").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("scope", "あいうえお").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	})

	t.Run("Found (no valid scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("scope", "write").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	})

	t.Run("Found (unknown response_type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "aiueo").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid response_type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code token none").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (prompt=none with no session)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errLoginRequired, loc.Query().Get("error"))
		}
	})

	t.Run("Found (prompt=none without consent)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			WithCookie(sessions.CookieName, session).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errConsentRequired, loc.Query().Get("error"))
		}
	})

	t.Run("Found (invalid prompt)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "ああああ").
			WithCookie(sessions.CookieName, session).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	})

	t.Run("Found (prompt=none with broader scope)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := mustMakeUser(t, repo, random)
		_, err := repo.IssueToken(client, user.ID, client.RedirectURI, scopesRead, 1000, false)
		require.NoError(t, err)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			WithFormField("scope", "read write").
			WithCookie(sessions.CookieName, generateSession(t, user.ID)).
			Expect()
		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
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
			ID:           utils.RandAlphabetAndNumberString(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       utils.RandAlphabetAndNumberString(36),
			Scopes:       scopes,
		}
		require.NoError(t, repo.SaveClient(client))

		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize").
			WithFormField("client_id", client.ID).
			Expect()
		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})
}

func TestHandlers_AuthorizationDecideHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common7)

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")
	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesRead,
	}
	require.NoError(t, repo.SaveClient(client))

	MakeDecideSession := func(t *testing.T, uid uuid.UUID, client *model.OAuth2Client) string {
		req := httptest.NewRequest(echo.GET, "/", nil)
		rec := httptest.NewRecorder()
		s, err := sessions.Get(rec, req, true)
		require.NoError(t, err)
		require.NoError(t, s.SetUser(uid))
		require.NoError(t, s.Set(oauth2ContextSession, authorizeRequest{
			ResponseType: "code",
			ClientID:     client.ID,
			RedirectURI:  client.RedirectURI,
			Scopes:       scopesReadWrite,
			ValidScopes:  scopesRead,
			State:        "state",
			Types:        responseType{true, false, false},
			AccessTime:   time.Now(),
			Nonce:        "nonce",
		}))

		return parseCookies(rec.Header().Get("Set-Cookie"))[sessions.CookieName].Value
	}

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.ID, client)).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal("state", loc.Query().Get("state"))
			assert.NotEmpty(loc.Query().Get("code"))
		}

		a, err := repo.GetAuthorize(loc.Query().Get("code"))
		if assert.NoError(err) {
			assert.Equal("nonce", a.Nonce)
		}
	})

	t.Run("Bad Request (No form)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.ID, client)).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Forbidden (No oauth2ContextSession)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, session).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (client not found)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.ID, &model.OAuth2Client{ID: "aaaa"})).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Forbidden (client without redirect uri", func(t *testing.T) {
		t.Parallel()
		client := &model.OAuth2Client{
			ID:           utils.RandAlphabetAndNumberString(36),
			Name:         "test client",
			Confidential: true,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       utils.RandAlphabetAndNumberString(36),
			Scopes:       scopesRead,
		}
		require.NoError(t, repo.SaveClient(client))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.ID, client)).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Found (deny)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "deny").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.ID, client)).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errAccessDenied, loc.Query().Get("error"))
		}
	})

	t.Run("Found (unsupported response type)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		req := httptest.NewRequest(echo.GET, "/", nil)
		rec := httptest.NewRecorder()
		s, err := sessions.Get(rec, req, true)
		require.NoError(t, err)
		require.NoError(t, s.SetUser(user.ID))
		require.NoError(t, s.Set(oauth2ContextSession, authorizeRequest{
			ResponseType: "code",
			ClientID:     client.ID,
			RedirectURI:  client.RedirectURI,
			Scopes:       scopesReadWrite,
			ValidScopes:  scopesRead,
			State:        "state",
			AccessTime:   time.Now(),
		}))

		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, parseCookies(rec.Header().Get("Set-Cookie"))[sessions.CookieName].Value).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	})

	t.Run("Found (timeout)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		req := httptest.NewRequest(echo.GET, "/", nil)
		rec := httptest.NewRecorder()
		s, err := sessions.Get(rec, req, true)
		require.NoError(t, err)
		require.NoError(t, s.SetUser(user.ID))
		require.NoError(t, s.Set(oauth2ContextSession, authorizeRequest{
			ResponseType: "code",
			ClientID:     client.ID,
			RedirectURI:  client.RedirectURI,
			Scopes:       scopesReadWrite,
			ValidScopes:  scopesRead,
			State:        "state",
			AccessTime:   time.Now().Add(-6 * time.Minute),
		}))

		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, parseCookies(rec.Header().Get("Set-Cookie"))[sessions.CookieName].Value).
			Expect()

		res.Status(http.StatusFound)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		loc, err := res.Raw().Location()
		if assert.NoError(err) {
			assert.Equal(errAccessDenied, loc.Query().Get("error"))
		}
	})
}

func TestHandlers_TokenEndpointHandler(t *testing.T) {
	t.Parallel()
	_, server, _, _, _, _ := setup(t, common7)

	t.Run("Unsupported Grant Type", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", "ああああ").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errUnsupportedGrantType)
	})
}

func TestHandlers_TokenEndpointClientCredentialsHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _ := setup(t, common7)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")
	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(client))

	t.Run("Success with Basic Auth", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.NotContainsKey("refresh_token")
		obj.Value("scope").String().Equal(client.Scopes.String())
	})

	t.Run("Success with form Auth", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("client_id", client.ID).
			WithFormField("client_secret", client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.NotContainsKey("refresh_token")
		obj.Value("scope").String().Equal(client.Scopes.String())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "read").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.NotContainsKey("refresh_token")
		obj.NotContainsKey("scope")
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "read manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.NotContainsKey("refresh_token")
		obj.Value("scope").String().Equal("read")
	})

	t.Run("Invalid Client (No credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Unknown client)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth("wrong client", "wrong password").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Not confidential)", func(t *testing.T) {
		t.Parallel()
		client := &model.OAuth2Client{
			ID:           utils.RandAlphabetAndNumberString(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       utils.RandAlphabetAndNumberString(36),
			RedirectURI:  "http://example.com",
			Scopes:       scopesReadWrite,
		}
		require.NoError(t, repo.SaveClient(client))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errUnauthorizedClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "アイウエオ").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeClientCredentials).
			WithFormField("scope", "manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointPasswordHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, user, _ := setupWithUsers(t, common7)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")
	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(client))

	t.Run("Success with Basic Auth", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal(client.Scopes.String())
	})

	t.Run("Success with form Auth", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("client_id", client.ID).
			WithFormField("client_secret", client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal(client.Scopes.String())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("scope", "read").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("scope", "read manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal("read")
	})

	t.Run("Success with not confidential client", func(t *testing.T) {
		t.Parallel()
		client := &model.OAuth2Client{
			ID:           utils.RandAlphabetAndNumberString(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       utils.RandAlphabetAndNumberString(36),
			RedirectURI:  "http://example.com",
			Scopes:       scopesReadWrite,
		}
		require.NoError(t, repo.SaveClient(client))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("client_id", client.ID).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal(client.Scopes.String())
	})

	t.Run("Invalid Request (No user credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidRequest)
	})

	t.Run("Invalid Grant (Wrong user credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "wrong password").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidGrant)
	})

	t.Run("Invalid Client (No client credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithBasicAuth(client.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Unknown client)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithBasicAuth("wrong client", "wrong password").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("scope", "ああああ").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypePassword).
			WithFormField("username", user.Name).
			WithFormField("password", "test").
			WithFormField("scope", "manage_bot").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointRefreshTokenHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, user, _ := setupWithUsers(t, common7)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")
	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(client))

	clientConf := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(clientConf))

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, client, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, client, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "read").
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal("read")

		_, err := repo.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, client, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "read manage_bot").
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal("read")

		_, err := repo.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client Basic Auth", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, clientConf, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client form Auth", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, clientConf, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("client_id", clientConf.ID).
			WithFormField("client_secret", clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetTokenByRefresh(token.RefreshToken)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (No refresh token)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidRequest)
	})

	t.Run("Invalid Grant (Unknown refresh token)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", "unknown token").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidGrant)
	})

	t.Run("Invalid Client (No client credentials)", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, clientConf, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, clientConf, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithBasicAuth(clientConf.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidClient)
	})

	t.Run("Invalid Scope (unknown scope)", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, client, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "アイウエオ").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})

	t.Run("Invalid Scope (no valid scope)", func(t *testing.T) {
		t.Parallel()
		token := mustIssueToken(t, repo, client, user.ID, true)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeRefreshToken).
			WithFormField("refresh_token", token.RefreshToken).
			WithFormField("scope", "manage_bot").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").String().Equal(errInvalidScope)
	})
}

func TestHandlers_TokenEndpointAuthorizationCodeHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, user, _ := setupWithUsers(t, common7)

	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")
	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")
	scopesReadManageBot := model.AccessScopes{}
	scopesReadManageBot.Add("read", "manage_bot")
	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(client))

	clientConf := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       utils.RandAlphabetAndNumberString(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesReadWrite,
	}
	require.NoError(t, repo.SaveClient(clientConf))

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		authorize := mustMakeAuthorizeData(t, repo, client.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("client_id", client.ID).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client Basic Auth", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with confidential client form Auth", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("client_id", clientConf.ID).
			WithFormField("client_secret", clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with PKCE(plain)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                utils.RandAlphabetAndNumberString(36),
			ClientID:            clientConf.ID,
			UserID:              user.ID,
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "plain",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with PKCE(S256)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                utils.RandAlphabetAndNumberString(36),
			ClientID:            clientConf.ID,
			UserID:              user.ID,
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "S256",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with smaller scope", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           utils.RandAlphabetAndNumberString(36),
			ClientID:       clientConf.ID,
			UserID:         user.ID,
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesRead,
			OriginalScopes: scopesRead,
			Nonce:          "nonce",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.NotContainsKey("scope")

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success with invalid scope", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           utils.RandAlphabetAndNumberString(36),
			ClientID:       client.ID,
			UserID:         user.ID,
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesRead,
			OriginalScopes: scopesReadManageBot,
			Nonce:          "nonce",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusOK)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		obj := res.JSON().Object()
		obj.Value("access_token").String().NotEmpty()
		obj.Value("token_type").String().Equal(authScheme)
		obj.Value("expires_in").Number().Equal(1000)
		obj.Value("refresh_token").String().NotEmpty()
		obj.Value("scope").String().Equal(authorize.Scopes.String())

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (No code)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidRequest)
	})

	t.Run("Invalid Client (No client)", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidClient)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (Wrong client credentials)", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, "wrong password").
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidClient)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (Other client)", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(client.ID, client.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidClient)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (Wrong code)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", "unknown").
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidGrant)
	})

	t.Run("Invalid Grant (expired)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           utils.RandAlphabetAndNumberString(36),
			ClientID:       clientConf.ID,
			UserID:         user.ID,
			CreatedAt:      time.Now(),
			ExpiresIn:      -1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidGrant)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Client (client not found)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           utils.RandAlphabetAndNumberString(36),
			ClientID:       utils.RandAlphabetAndNumberString(36),
			UserID:         user.ID,
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			RedirectURI:    "http://example.com",
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidClient)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (different redirect)", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example2.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidGrant)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Grant (unexpected redirect)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:           utils.RandAlphabetAndNumberString(36),
			ClientID:       clientConf.ID,
			UserID:         user.ID,
			CreatedAt:      time.Now(),
			ExpiresIn:      1000,
			Scopes:         scopesReadWrite,
			OriginalScopes: scopesReadWrite,
			Nonce:          "nonce",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusUnauthorized)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidGrant)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (PKCE failure)", func(t *testing.T) {
		t.Parallel()
		authorize := &model.OAuth2Authorize{
			Code:                utils.RandAlphabetAndNumberString(36),
			ClientID:            clientConf.ID,
			UserID:              user.ID,
			CreatedAt:           time.Now(),
			ExpiresIn:           1000,
			RedirectURI:         "http://example.com",
			Scopes:              scopesReadWrite,
			OriginalScopes:      scopesReadWrite,
			Nonce:               "nonce",
			CodeChallengeMethod: "plain",
			CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		}
		require.NoError(t, repo.SaveAuthorize(authorize))
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidRequest)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Invalid Request (unexpected PKCE)", func(t *testing.T) {
		t.Parallel()
		authorize := mustMakeAuthorizeData(t, repo, clientConf.ID, user.ID)
		e := makeExp(t, server)
		res := e.POST("/api/1.0/oauth2/token").
			WithFormField("grant_type", grantTypeAuthorizationCode).
			WithFormField("code", authorize.Code).
			WithFormField("redirect_uri", "http://example.com").
			WithFormField("code_verifier", "jfeiajoijioajfoiwjo").
			WithBasicAuth(clientConf.ID, clientConf.Secret).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
		res.JSON().Object().Value("error").Equal(errInvalidRequest)

		_, err := repo.GetAuthorize(authorize.Code)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}

func TestHandlers_RevokeTokenEndpointHandler(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, user, _ := setupWithUsers(t, common6)

	t.Run("NoToken", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/oauth2/revoke").
			WithFormField("token", "").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("AccessToken", func(t *testing.T) {
		t.Parallel()
		token, err := repo.IssueToken(nil, user.ID, "", model.AccessScopes{}, 10000, false)
		require.NoError(t, err)

		e := makeExp(t, server)
		e.POST("/api/1.0/oauth2/revoke").
			WithFormField("token", token.AccessToken).
			Expect().
			Status(http.StatusOK)

		_, err = repo.GetTokenByID(token.ID)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("RefreshToken", func(t *testing.T) {
		t.Parallel()
		token, err := repo.IssueToken(nil, user.ID, "", model.AccessScopes{}, 10000, true)
		require.NoError(t, err)

		e := makeExp(t, server)
		e.POST("/api/1.0/oauth2/revoke").
			WithFormField("token", token.RefreshToken).
			Expect().
			Status(http.StatusOK)

		_, err = repo.GetTokenByID(token.ID)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})
}
