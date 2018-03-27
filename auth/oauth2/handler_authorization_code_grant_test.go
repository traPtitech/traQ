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
	"time"
)

// Authorization Code Grantのテスト

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
