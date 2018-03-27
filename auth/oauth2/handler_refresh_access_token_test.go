package oauth2

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Refreshing an Access Tokenのテスト

func BeforeTestRefreshAccessToken(t *testing.T) (*assert.Assertions, *require.Assertions, *Handler, *Client, *Token, *echo.Echo) {
	assert := assert.New(t)
	require := require.New(t)
	store := NewStoreMock()

	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: false,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
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
	}

	token, err := handler.IssueAccessToken(client, uuid.NewV4(), client.RedirectURI, client.Scopes, handler.AccessTokenExp, true)
	require.NoError(err)

	return assert, require, handler, client, token, e
}

func PostTestRefreshAccessTokenSuccess(assert *assert.Assertions, h *Handler, req *http.Request, e *echo.Echo) {
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
// コンフィデンシャルでないクライアント
func TestRefreshAccessToken_Success1(t *testing.T) {
	t.Parallel()

	assert, _, h, _, token, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)

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

	PostTestRefreshAccessTokenSuccess(assert, h, req, e)
}

// 成功パターン2
// 要求スコープを縮小
func TestRefreshAccessToken_Success2(t *testing.T) {
	t.Parallel()

	assert, _, h, _, token, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)
	f.Set("scope", "private_read")

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
			assert.Equal("private_read", res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestRefreshAccessTokenSuccess(assert, h, req, e)
}

// 成功パターン3
// 無効な要求スコープを含む
func TestRefreshAccessToken_Success3(t *testing.T) {
	t.Parallel()

	assert, _, h, _, token, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)
	f.Set("scope", "private_read write")

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
			assert.Equal("private_read", res.Scope)
			assert.Empty(res.IDToken)
		}
	}

	PostTestRefreshAccessTokenSuccess(assert, h, req, e)
}

// 成功パターン4
// コンフィデンシャルなクライアント(Basic認証)
func TestRefreshAccessToken_Success4(t *testing.T) {
	t.Parallel()

	assert, require, h, _, _, e := BeforeTestRefreshAccessToken(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))
	token, err := h.IssueAccessToken(client, uuid.NewV4(), client.RedirectURI, client.Scopes, h.AccessTokenExp, true)
	require.NoError(err)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)

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

	PostTestRefreshAccessTokenSuccess(assert, h, req, e)
}

// 成功パターン5
// コンフィデンシャルなクライアント(Basic認証を用いない)
func TestRefreshAccessToken_Success5(t *testing.T) {
	t.Parallel()

	assert, require, h, _, _, e := BeforeTestRefreshAccessToken(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))
	token, err := h.IssueAccessToken(client, uuid.NewV4(), client.RedirectURI, client.Scopes, h.AccessTokenExp, true)
	require.NoError(err)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)
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

	PostTestRefreshAccessTokenSuccess(assert, h, req, e)
}

// 失敗パターン1
// リフレッシュトークンがない
func TestRefreshAccessToken_Failure1(t *testing.T) {
	t.Parallel()

	assert, _, h, _, _, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)

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
			assert.Equal(errInvalidRequest, res.ErrorType)
		}
	}
}

// 失敗パターン2
// 存在しないリフレッシュトークン
func TestRefreshAccessToken_Failure2(t *testing.T) {
	t.Parallel()

	assert, _, h, _, _, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", "存在しない")

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
			assert.Equal(errInvalidGrant, res.ErrorType)
		}
	}
}

// 失敗パターン3
// 要求スコープが不正
func TestRefreshAccessToken_Failure3(t *testing.T) {
	t.Parallel()

	assert, _, h, _, token, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)
	f.Set("scope", "アイウエオ")

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
			assert.Equal(errInvalidScope, res.ErrorType)
		}
	}
}

// 失敗パターン4
// 要求スコープが全て無効
func TestRefreshAccessToken_Failure4(t *testing.T) {
	t.Parallel()

	assert, _, h, _, token, e := BeforeTestRefreshAccessToken(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)
	f.Set("scope", "write")

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
			assert.Equal(errInvalidScope, res.ErrorType)
		}
	}
}

// 失敗パターン5
// コンフィデンシャルなクライアントの認証情報がない
func TestRefreshAccessToken_Failure5(t *testing.T) {
	t.Parallel()

	assert, require, h, _, _, e := BeforeTestRefreshAccessToken(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))
	token, err := h.IssueAccessToken(client, uuid.NewV4(), client.RedirectURI, client.Scopes, h.AccessTokenExp, true)
	require.NoError(err)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)

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
}

// 失敗パターン6
// コンフィデンシャルなクライアントの認証情報が間違っている
func TestRefreshAccessToken_Failure6(t *testing.T) {
	t.Parallel()

	assert, require, h, _, _, e := BeforeTestRefreshAccessToken(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client",
		Confidential: true,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
			scope.PrivateRead,
		},
	}
	require.NoError(h.SaveClient(client))
	token, err := h.IssueAccessToken(client, uuid.NewV4(), client.RedirectURI, client.Scopes, h.AccessTokenExp, true)
	require.NoError(err)

	f := url.Values{}
	f.Set("grant_type", grantTypeRefreshToken)
	f.Set("refresh_token", token.RefreshToken)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, "間違ってるよ")

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
}
