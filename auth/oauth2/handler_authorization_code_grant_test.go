package oauth2

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/quasoft/memstore"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Authorization Code Grantのテスト

// AuthorizationEndpointHandler

func BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t *testing.T) (*assert.Assertions, *require.Assertions, *Handler, *Client, *echo.Echo) {
	assert := assert.New(t)
	require := require.New(t)
	store := NewStoreMock()

	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.OpenID,
			scope.Profile,
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(store.SaveClient(client))

	e := echo.New()
	handler := &Handler{
		Store:                store,
		AccessTokenExp:       1000,
		AuthorizationCodeExp: 1000,
		IsRefreshEnabled:     true,
		Sessions:             memstore.NewMemStore([]byte("secret")),
		UserInfoGetter: func(uid uuid.UUID) (UserInfo, error) {
			if uid == uuid.Nil {
				return nil, ErrUserIDOrPasswordWrong
			}
			return &UserInfoMock{uid: uid}, nil
		},
	}

	return assert, require, handler, client, e
}

func MakeSession(t *testing.T, h *Handler, uid uuid.UUID) *http.Cookie {
	req := httptest.NewRequest(echo.GET, "/", nil)
	rec := httptest.NewRecorder()
	s, err := h.Sessions.New(req, "sessions")
	require.NoError(t, err)
	s.Values["userID"] = uid.String()
	require.NoError(t, s.Save(req, rec))

	return parseCookies(rec.Header().Get("Set-Cookie"))["sessions"]
}

func parseCookies(value string) map[string]*http.Cookie {
	m := map[string]*http.Cookie{}
	for _, c := range (&http.Request{Header: http.Header{"Cookie": {value}}}).Cookies() {
		m[c.Name] = c
	}
	return m
}

// 成功パターン1
// prompt=none
func TestAuthorizationCodeGrantAuthorizationEndpoint_Success1(t *testing.T) {
	t.Parallel()

	assert, require, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)
	uid := uuid.NewV4()
	_, err := h.IssueAccessToken(client, uid, client.RedirectURI, scope.AccessScopes{scope.Read}, 10000, false)
	require.NoError(err)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("state", "state")
	f.Set("prompt", "none")
	f.Set("scope", "read")
	f.Set("nonce", "nonce")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(MakeSession(t, h, uid))

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(f.Get("state"), loc.Query().Get("state"))
			assert.NotEmpty(loc.Query().Get("code"))
		}

		a, err := h.GetAuthorize(loc.Query().Get("code"))
		if assert.NoError(err) {
			assert.Equal(f.Get("nonce"), a.Nonce)
		}
	}
}

// 成功パターン2
// code
func TestAuthorizationCodeGrantAuthorizationEndpoint_Success2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("state", "state")
	f.Set("nonce", "nonce")
	f.Set("scope", "read write")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(f.Get("state"), loc.Query().Get("state"))
			assert.Equal(f.Get("client_id"), loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		s, err := h.Sessions.Get(req, "sessions")
		if assert.NoError(err) {
			assert.Equal(f.Get("state"), s.Values[oauth2ContextSession].(authorizeRequest).State)
		}
	}
}

// 成功パターン3
// pkceつき
func TestAuthorizationCodeGrantAuthorizationEndpoint_Success3(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("state", "state")
	f.Set("nonce", "nonce")
	f.Set("scope", "read write")
	f.Set("code_challenge_method", "S256")
	f.Set("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(f.Get("state"), loc.Query().Get("state"))
			assert.Equal(f.Get("client_id"), loc.Query().Get("client_id"))
			assert.Equal("read", loc.Query().Get("scopes"))
		}

		s, err := h.Sessions.Get(req, "sessions")
		if assert.NoError(err) {
			assert.Equal(f.Get("state"), s.Values[oauth2ContextSession].(authorizeRequest).State)
			assert.Equal(f.Get("code_challenge"), s.Values[oauth2ContextSession].(authorizeRequest).CodeChallenge)
		}
	}
}

// 失敗パターン1
// リクエストが不正
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure1(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	req := httptest.NewRequest(echo.POST, "/", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.Error(h.AuthorizationEndpointHandler(c)) {
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
	}
}

// 失敗パターン2
// クライアントIDがない
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure2(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.EqualError(echo.NewHTTPError(http.StatusBadRequest), h.AuthorizationEndpointHandler(c).Error()) {
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
	}
}

// 失敗パターン3
// 存在しないクライアント
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure3(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", "存在しない")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.EqualError(echo.NewHTTPError(http.StatusBadRequest), h.AuthorizationEndpointHandler(c).Error()) {
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
	}
}

// 失敗パターン4
// クライアントにリダイレクトURIが設定されていない
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure4(t *testing.T) {
	t.Parallel()

	assert, require, h, _, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		Scopes: scope.AccessScopes{
			scope.OpenID,
			scope.Profile,
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))

	f := url.Values{}
	f.Set("client_id", client.ID)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.EqualError(echo.NewHTTPError(http.StatusForbidden), h.AuthorizationEndpointHandler(c).Error()) {
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
	}
}

// 失敗パターン5
// リダイレクトURIが異なる
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure5(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("redirect_uri", "ちがう")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.EqualError(echo.NewHTTPError(http.StatusBadRequest), h.AuthorizationEndpointHandler(c).Error()) {
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
	}
}

// 失敗パターン6
// PKCEのメソッドが不正
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure6(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("code_challenge_method", "aiueo")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン7
// PKCEのチャレンジが不正
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure7(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("code_challenge_method", "S256")
	f.Set("code_challenge", "ああああ")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン8
// 要求スコープが不正
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure8(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("scope", "あいうえお")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン9
// 要求スコープが全て無効
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure9(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("scope", "write")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidScope, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン10
// 不明なResponseType
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure10(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "aiueo")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン11
// 不正なResponseType
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure11(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code none")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errUnsupportedResponseType, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン12
// prompt=noneでログインしていない
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure12(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("prompt", "none")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errLoginRequired, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン13
// prompt=noneで以前に許可されていない
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure13(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("prompt", "none")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(MakeSession(t, h, uuid.NewV4()))

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errConsentRequired, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン14
// prompt=noneで以前に許可されているが、スコープが広がっている
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure14(t *testing.T) {
	t.Parallel()

	assert, require, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)
	uid := uuid.NewV4()
	_, err := h.IssueAccessToken(client, uid, client.RedirectURI, scope.AccessScopes{scope.Read}, 10000, false)
	require.NoError(err)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("prompt", "none")
	f.Set("scope", "read private_read")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(MakeSession(t, h, uid))

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errConsentRequired, loc.Query().Get("error"))
		}
	}
}

// 失敗パターン15
// サポートしないprompt
func TestAuthorizationCodeGrantAuthorizationEndpoint_Failure15(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestAuthorizationCodeGrantAuthorizationEndpoint(t)

	f := url.Values{}
	f.Set("client_id", client.ID)
	f.Set("response_type", "code")
	f.Set("prompt", "ああああ")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.AuthorizationEndpointHandler(c)) {
		assert.Equal(http.StatusFound, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))
		loc, err := rec.Result().Location()
		if assert.NoError(err) {
			assert.Equal(errInvalidRequest, loc.Query().Get("error"))
		}
	}
}

// TokenEndpointHandler
// TODO IDToken発行テスト

func BeforeTestAuthorizationCodeGrantTokenEndpoint(t *testing.T) (*assert.Assertions, *require.Assertions, *Handler, *Client, *AuthorizeData, *echo.Echo) {
	assert := assert.New(t)
	require := require.New(t)
	store := NewStoreMock()

	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.OpenID,
			scope.Profile,
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(store.SaveClient(client))
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(store.SaveAuthorize(authorize))

	e := echo.New()
	handler := &Handler{
		Store:                store,
		AccessTokenExp:       1000,
		AuthorizationCodeExp: 1000,
		IsRefreshEnabled:     true,
	}

	return assert, require, handler, client, authorize, e

}

func PostTestAuthorizationCodeGrantTokenEndpoint(assert *assert.Assertions, h *Handler, req *http.Request, e *echo.Echo) {
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 2度目は無効
	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}
}

// 成功パターン1
// コンフィデンシャルなクライアントでBasic認証を用いる
func TestAuthorizationCodeGrantTokenEndpoint_Success1(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン2
// コンフィデンシャルなクライアントでBasic認証を用いない
func TestAuthorizationCodeGrantTokenEndpoint_Success2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")
	f.Set("client_id", client.ID)
	f.Set("client_secret", client.Secret)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン3
// PKCEを使用(plain)
func TestAuthorizationCodeGrantTokenEndpoint_Success3(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce:               "nonce",
		CodeChallengeMethod: "plain",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")
	f.Set("code_verifier", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン4
// PKCEを使用(S256)
func TestAuthorizationCodeGrantTokenEndpoint_Success4(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce:               "nonce",
		CodeChallengeMethod: "S256",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")
	f.Set("code_verifier", "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン5
// 要求スコープを縮小
func TestAuthorizationCodeGrantTokenEndpoint_Success5(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン6
// 無効な要求スコープを含む
func TestAuthorizationCodeGrantTokenEndpoint_Success6(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.PrivateRead,
			scope.Write,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Equal(authorize.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 成功パターン7
// コンフィデンシャルでないクライアント
func TestAuthorizationCodeGrantTokenEndpoint_Success7(t *testing.T) {
	t.Parallel()

	assert, require, h, _, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.OpenID,
			scope.Profile,
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")
	f.Set("client_id", client.ID)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusOK, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := tokenResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.NotEmpty(res.AccessToken)
			assert.Equal(AuthScheme, res.TokenType)
			assert.Equal(1000, res.ExpiresIn)
			assert.NotEmpty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン1
// 認可コードがない
func TestAuthorizationCodeGrantTokenEndpoint_Failure1(t *testing.T) {
	t.Parallel()

	assert, _, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidRequest, res.ErrorType)
		}
	}
}

// 失敗パターン2
// クライアントの認証情報がない
func TestAuthorizationCodeGrantTokenEndpoint_Failure2(t *testing.T) {
	t.Parallel()

	assert, _, h, _, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidClient, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン3
// コンフィデンシャルなクライアントの認証に失敗
func TestAuthorizationCodeGrantTokenEndpoint_Failure3(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, "間違っている")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusUnauthorized, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidClient, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン4
// 認可コードのクライアントと異なる
func TestAuthorizationCodeGrantTokenEndpoint_Failure4(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth("違う", client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusUnauthorized, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidClient, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン5
// 認可コードが間違っている
func TestAuthorizationCodeGrantTokenEndpoint_Failure5(t *testing.T) {
	t.Parallel()

	assert, _, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", "ないよ")
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}
}

// 失敗パターン6
// 認可コードの有効期限が切れている
func TestAuthorizationCodeGrantTokenEndpoint_Failure6(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   -1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン7
// 認可コードのクライアントが存在しない
func TestAuthorizationCodeGrantTokenEndpoint_Failure7(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    generateRandomString(),
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidClient, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン8
// リダイレクトURIが異なる
func TestAuthorizationCodeGrantTokenEndpoint_Failure8(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example2.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusUnauthorized, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン9
// 認可エンドポイントでリダイレクトURIを指定していないのにリダイレクトURIが存在
func TestAuthorizationCodeGrantTokenEndpoint_Failure9(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:      generateRandomString(),
		ClientID:  client.ID,
		UserID:    uuid.NewV4(),
		CreatedAt: time.Now(),
		ExpiresIn: 1000,
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce: "nonce",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusUnauthorized, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン10
// PKCEのバリデーションに失敗
func TestAuthorizationCodeGrantTokenEndpoint_Failure10(t *testing.T) {
	t.Parallel()

	assert, require, h, client, _, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)
	authorize := &AuthorizeData{
		Code:        generateRandomString(),
		ClientID:    client.ID,
		UserID:      uuid.NewV4(),
		CreatedAt:   time.Now(),
		ExpiresIn:   1000,
		RedirectURI: "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		OriginalScopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
		Nonce:               "nonce",
		CodeChallengeMethod: "plain",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
	}
	require.NoError(h.SaveAuthorize(authorize))

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidRequest, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}

// 失敗パターン11
// PKCEを使用していないのにcode_verifierが存在
func TestAuthorizationCodeGrantTokenEndpoint_Failure11(t *testing.T) {
	t.Parallel()

	assert, _, h, client, authorize, e := BeforeTestAuthorizationCodeGrantTokenEndpoint(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeAuthorizationCode)
	f.Set("code", authorize.Code)
	f.Set("redirect_uri", "http://example.com")
	f.Set("code_verifier", "jfeiajoijioajfoiwjo")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, client.Secret)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidRequest, res.ErrorType)
		}
	}

	PostTestAuthorizationCodeGrantTokenEndpoint(assert, h, req, e)
}
