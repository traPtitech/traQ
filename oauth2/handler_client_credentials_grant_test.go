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

// Client Credentials Grantのテスト

func BeforeTestClientCredentialsGrant(t *testing.T) (*assert.Assertions, *require.Assertions, *Handler, *Client, *echo.Echo) {
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
	}

	return assert, require, handler, client, e
}

// 成功パターン1
// Basic認証を用いる
func TestClientCredentialsGrant_Success1(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)

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
			assert.Empty(res.RefreshToken)
			assert.Equal(client.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン2
// Basic認証を用いない
func TestClientCredentialsGrant_Success2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)
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
			assert.Empty(res.RefreshToken)
			assert.Equal(client.Scopes.String(), res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン3
// 要求スコープを縮小
func TestClientCredentialsGrant_Success3(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)
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
			assert.Empty(res.RefreshToken)
			assert.Empty(res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 成功パターン4
// 無効な要求スコープを含む
func TestClientCredentialsGrant_Success4(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)
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
			assert.Empty(res.RefreshToken)
			assert.Equal("private_read", res.Scope)
			assert.Empty(res.IDToken)
		}
	}
}

// 失敗パターン1
// クライアント認証情報がない
func TestClientCredentialsGrant_Failure1(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)

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

// 失敗パターン2
// クライアント認証情報が間違っている
func TestClientCredentialsGrant_Failure2(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth(client.ID, "間違ってますよ")

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

// 失敗パターン3
// 存在しないクライアント
func TestClientCredentialsGrant_Failure3(t *testing.T) {
	t.Parallel()

	assert, _, h, _, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.SetBasicAuth("ないよ", "間違ってますよ")

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

// 失敗パターン4
// コンフィデンシャルでないクライアント
func TestClientCredentialsGrant_Failure4(t *testing.T) {
	t.Parallel()

	assert, require, h, _, e := BeforeTestClientCredentialsGrant(t)
	client := &Client{
		ID:           generateRandomString(),
		Name:         "test client2",
		Confidential: false,
		CreatorID:    uuid.NewV4(),
		Secret:       generateRandomString(),
		RedirectURI:  "http://example.com",
		Scopes: scope.AccessScopes{
			scope.Read,
		},
	}
	require.NoError(h.SaveClient(client))

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)

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
			assert.Equal(errUnauthorizedClient, res.ErrorType)
		}
	}
}

// 失敗パターン5
// 要求スコープが不正
func TestClientCredentialsGrant_Failure5(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)
	f.Set("scope", "アイウエオ")

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

// 失敗パターン6
// 要求スコープが全て無効
func TestClientCredentialsGrant_Failure6(t *testing.T) {
	t.Parallel()

	assert, _, h, client, e := BeforeTestClientCredentialsGrant(t)

	f := url.Values{}
	f.Set("grant_type", grantTypeClientCredentials)
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
