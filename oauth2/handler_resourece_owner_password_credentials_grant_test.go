package oauth2

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/oauth2/scope"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Resource Owner Password Credentials Grantのテスト

func BeforeTestResourceOwnerPasswordCredentialsGrant(t *testing.T) (*assert.Assertions, *require.Assertions, *Handler, *Client, *echo.Echo) {
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
		UserAuthenticator: func(id, pw string) (uuid.UUID, error) {
			if id == "test" && pw == "test" {
				return uuid.NewV4(), nil
			}
			return uuid.Nil, ErrUserIDOrPasswordWrong
		},
	}

	return assert, require, handler, client, e
}

// 成功パターン1
// クレデンシャルなクライアントでBasic認証を用いる
func TestResourceOwnerPasswordCredentialsGrant_Success1(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")

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
			assert.Equal(client.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン2
// クレデンシャルなクライアントでBasic認証を用いない
func TestResourceOwnerPasswordCredentialsGrant_Success2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
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
			assert.Equal(client.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン3
// 要求スコープを縮小
func TestResourceOwnerPasswordCredentialsGrant_Success3(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
	f.Set("scope", "private_read")

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
}

// 成功パターン4
// 無効な要求スコープを含む
func TestResourceOwnerPasswordCredentialsGrant_Success4(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
	f.Set("scope", "private_read write")

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
			assert.Equal("private_read", res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン5
// クレデンシャルなクライアントでない
func TestResourceOwnerPasswordCredentialsGrant_Success5(t *testing.T) {
	t.Parallel()

	assert, require, h, _, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)
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
	require.NoError(h.SaveClient(client))

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
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
			assert.Equal(client.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 失敗パターン1
// ユーザー認証情報がない
func TestResourceOwnerPasswordCredentialsGrant_Failure1(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)

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
// ユーザー認証に失敗
func TestResourceOwnerPasswordCredentialsGrant_Failure2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "あああ")

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
}

// 失敗パターン3
// クレデンシャルなクライアントの認証に失敗
func TestResourceOwnerPasswordCredentialsGrant_Failure3(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")

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
}

// 失敗パターン4
// クライアント情報がない
func TestResourceOwnerPasswordCredentialsGrant_Failure4(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")

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

// 失敗パターン5
// 存在しないクライアント
func TestResourceOwnerPasswordCredentialsGrant_Failure5(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth("存在しない", "間違っている")

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
// 要求スコープが不正
func TestResourceOwnerPasswordCredentialsGrant_Failure6(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
	f.Set("scope", "ああああ")

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
			assert.Equal(errInvalidScope, res.ErrorType)
		}
	}
}

// 失敗パターン7
// 要求スコープが全て無効
func TestResourceOwnerPasswordCredentialsGrant_Failure7(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestResourceOwnerPasswordCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypePassword)
	f.Set("username", "test")
	f.Set("password", "test")
	f.Set("scope", "write")

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
			assert.Equal(errInvalidScope, res.ErrorType)
		}
	}
}
