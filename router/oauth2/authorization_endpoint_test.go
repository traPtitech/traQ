package oauth2

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/sessions"
	random2 "github.com/traPtitech/traQ/utils/random"
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
	repo, server := Setup(t, db2)
	defaultUser := CreateUser(t, repo, rand)
	session := S(t, defaultUser.GetID())

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       random2.AlphaNumeric(36),
		RedirectURI:  "http://example.com",
		Scopes:       scopesRead,
	}
	require.NoError(t, repo.SaveClient(client))

	t.Run("Success (prompt=none)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		user := CreateUser(t, repo, rand)
		IssueToken(t, repo, client, user.GetID(), false)
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("state", "state").
			WithFormField("prompt", "none").
			WithFormField("scope", "read").
			WithFormField("nonce", "nonce").
			WithCookie(sessions.CookieName, S(t, user.GetID())).
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.GET("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (no client)", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", "").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (unknown client)", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", "unknown").
			Expect()
		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (different redirect uri)", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
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
		user := CreateUser(t, repo, rand)
		_, err := repo.IssueToken(client, user.GetID(), client.RedirectURI, scopesRead, 1000, false)
		require.NoError(t, err)
		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			WithFormField("response_type", "code").
			WithFormField("prompt", "none").
			WithFormField("scope", "read write").
			WithCookie(sessions.CookieName, S(t, user.GetID())).
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
			ID:           random2.AlphaNumeric(36),
			Name:         "test client",
			Confidential: false,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       random2.AlphaNumeric(36),
			Scopes:       scopes,
		}
		require.NoError(t, repo.SaveClient(client))

		e := R(t, server)
		res := e.POST("/oauth2/authorize").
			WithFormField("client_id", client.ID).
			Expect()
		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})
}

func TestHandlers_AuthorizationDecideHandler(t *testing.T) {
	t.Parallel()
	repo, server := Setup(t, db2)
	user := CreateUser(t, repo, rand)
	session := S(t, user.GetID())

	scopesRead := model.AccessScopes{}
	scopesRead.Add("read")
	scopesReadWrite := model.AccessScopes{}
	scopesReadWrite.Add("read", "write")

	client := &model.OAuth2Client{
		ID:           random2.AlphaNumeric(36),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.Must(uuid.NewV4()),
		Secret:       random2.AlphaNumeric(36),
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.GetID(), client)).
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
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Forbidden (No oauth2ContextSession)", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, session).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Bad Request (client not found)", func(t *testing.T) {
		t.Parallel()
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.GetID(), &model.OAuth2Client{ID: "aaaa"})).
			Expect()

		res.Status(http.StatusBadRequest)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Forbidden (client without redirect uri", func(t *testing.T) {
		t.Parallel()
		client := &model.OAuth2Client{
			ID:           random2.AlphaNumeric(36),
			Name:         "test client",
			Confidential: true,
			CreatorID:    uuid.Must(uuid.NewV4()),
			Secret:       random2.AlphaNumeric(36),
			Scopes:       scopesRead,
		}
		require.NoError(t, repo.SaveClient(client))
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "approve").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.GetID(), client)).
			Expect()

		res.Status(http.StatusForbidden)
		res.Header("Cache-Control").Equal("no-store")
		res.Header("Pragma").Equal("no-cache")
	})

	t.Run("Found (deny)", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
			WithFormField("submit", "deny").
			WithCookie(sessions.CookieName, MakeDecideSession(t, user.GetID(), client)).
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
		require.NoError(t, s.SetUser(user.GetID()))
		require.NoError(t, s.Set(oauth2ContextSession, authorizeRequest{
			ResponseType: "code",
			ClientID:     client.ID,
			RedirectURI:  client.RedirectURI,
			Scopes:       scopesReadWrite,
			ValidScopes:  scopesRead,
			State:        "state",
			AccessTime:   time.Now(),
		}))

		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
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
		require.NoError(t, s.SetUser(user.GetID()))
		require.NoError(t, s.Set(oauth2ContextSession, authorizeRequest{
			ResponseType: "code",
			ClientID:     client.ID,
			RedirectURI:  client.RedirectURI,
			Scopes:       scopesReadWrite,
			ValidScopes:  scopesRead,
			State:        "state",
			AccessTime:   time.Now().Add(-6 * time.Minute),
		}))

		e := R(t, server)
		res := e.POST("/oauth2/authorize/decide").
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
