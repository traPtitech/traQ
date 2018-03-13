package oauth2

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/openid"
	"github.com/traPtitech/traQ/auth/scope"
	"github.com/traPtitech/traQ/model"
	"net/http"
	"time"
)

const (
	// Authorization Code Grant
	grantTypeAuthorizationCode = "authorization_code"
	// Resource Owner Password Credentials Grant
	grantTypePassword = "password"
	// Client Credentials Grant
	grantTypeClientCredentials = "client_credentials"
	// Refreshing an Access Token
	grantTypeRefreshToken = "refresh_token"
)

// Token : OAuth2.0 Access Token構造体
type Token struct {
	ID           string
	ClientID     string
	UserID       uuid.UUID
	RedirectURI  string
	AccessToken  string
	RefreshToken string
	CreatedAt    time.Time
	ExpiresIn    int
	Scopes       scope.AccessScopes
}

type tokenRequest struct {
	GrantType    string `form:"grant_type"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id"`
	CodeVerifier string `form:"code_verifier"`
	Username     string `form:"username"`
	Password     string `form:"password"`
	Scope        string `form:"scope"`
	RefreshToken string `form:"refresh_token"`
	ClientSecret string `form:"client_secret"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// GetAvailableScopes : requestで与えられたスコープのうち、利用可能なものを返します
func (t *Token) GetAvailableScopes(request scope.AccessScopes) (result scope.AccessScopes) {
	for _, s := range request {
		if t.Scopes.Contains(s) {
			result = append(result, s)
		}
	}
	return
}

// TokenEndpointHandler : トークンエンドポイントのハンドラ
func TokenEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := &tokenRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidRequest})
	}

	getCIDAndCPW := func() (id, pw string, err error) {
		id, pw, ok := c.Request().BasicAuth()
		if !ok { // Request Body
			if len(req.ClientID) == 0 {
				return "", "", &errorResponse{ErrorType: errInvalidClient}
			}
			id = req.ClientID
			pw = req.Password
		}
		return
	}
	res := &tokenResponse{
		TokenType: "Bearer",
	}

	switch req.GrantType {
	case grantTypeAuthorizationCode:
		if len(req.Code) == 0 {
			return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidRequest})
		}

		// 認可コード確認
		code, err := store.GetAuthorize(req.Code)
		if err != nil {
			switch err {
			case ErrAuthorizeNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		// 認可コードは２回使えない
		if err := store.DeleteAuthorize(code.Code); err != nil {
			c.Logger().Error(err)
		}
		if code.IsExpired() {
			return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidGrant})
		}

		// クライアント確認
		client, err := store.GetClient(code.ClientID)
		if err != nil {
			switch err {
			case ErrClientNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		id, pw, err := getCIDAndCPW()
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		if client.ID != id || (client.Confidential && client.Secret != pw) {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidClient})
		}

		// リダイレクトURI確認
		if (len(code.RedirectURI) > 0 && client.RedirectURI != req.RedirectURI) || (len(code.RedirectURI) == 0 && len(req.RedirectURI) > 0) {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidGrant})
		}

		// PKCE確認
		if ok, _ := code.ValidatePKCE(req.CodeVerifier); !ok {
			return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidRequest})
		}

		// トークン発行
		newToken, err := IssueAccessToken(client, code.UserID, client.RedirectURI, code.Scopes, AccessTokenExp, IsRefreshEnabled)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		// OpenID Connect IDToken発行
		if code.Scopes.Contains(scope.OpenID) && openid.Available() {
			idToken := openid.NewIDToken(newToken.CreatedAt, int64(AccessTokenExp))
			idToken.Audience = code.ClientID
			idToken.Subject = code.UserID.String()
			idToken.Nonce = code.Nonce

			user, err := model.GetUser(code.UserID.String())
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			if code.Scopes.Contains(scope.Profile) {
				idToken.Name = user.Name
			}

			res.IDToken, err = idToken.Generate()
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

		res.AccessToken = newToken.AccessToken
		res.RefreshToken = newToken.RefreshToken
		res.ExpiresIn = newToken.ExpiresIn
		if len(code.OriginalScopes) != len(newToken.Scopes) {
			res.Scope = newToken.Scopes.String()
		}

	case grantTypePassword:
		cid, cpw, err := getCIDAndCPW()
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		// クライアント確認
		client, err := store.GetClient(cid)
		if err != nil {
			switch err {
			case ErrClientNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		if client.Confidential && client.Secret != cpw {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidClient})
		}

		// ユーザー確認
		if len(req.Username) == 0 {
			return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidRequest})
		}
		user := &model.User{Name: req.Username}
		if err := user.Authorization(req.Password); err != nil {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidGrant})
		}

		// 要求スコープ確認
		reqScopes, err := SplitAndValidateScope(req.Scope)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		validScopes := client.GetAvailableScopes(reqScopes)
		if len(reqScopes) == 0 {
			validScopes = client.Scopes
		} else if len(validScopes) == 0 {
			return c.JSON(http.StatusBadRequest, ErrInvalidScope)
		}

		// トークン発行
		newToken, err := IssueAccessToken(client, uuid.FromStringOrNil(user.ID), client.RedirectURI, validScopes, AccessTokenExp, IsRefreshEnabled)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		res.AccessToken = newToken.AccessToken
		res.RefreshToken = newToken.RefreshToken
		res.ExpiresIn = newToken.ExpiresIn
		if len(reqScopes) != len(validScopes) {
			res.Scope = newToken.Scopes.String()
		}

	case grantTypeClientCredentials:
		id, pw, err := getCIDAndCPW()
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}

		// クライアント確認
		client, err := store.GetClient(id)
		if err != nil {
			switch err {
			case ErrClientNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		if !client.Confidential {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errUnauthorizedClient})
		}
		if client.Secret != pw {
			return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidClient})
		}

		// 要求スコープ確認
		reqScopes, err := SplitAndValidateScope(req.Scope)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		validScopes := client.GetAvailableScopes(reqScopes)
		if len(reqScopes) == 0 {
			validScopes = client.Scopes
		} else if len(validScopes) == 0 {
			return c.JSON(http.StatusBadRequest, ErrInvalidScope)
		}

		// トークン発行
		newToken, err := IssueAccessToken(client, uuid.Nil, client.RedirectURI, validScopes, AccessTokenExp, false)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		res.AccessToken = newToken.AccessToken
		res.ExpiresIn = newToken.ExpiresIn
		if len(reqScopes) != len(validScopes) {
			res.Scope = newToken.Scopes.String()
		}

	case grantTypeRefreshToken:
		if len(req.RefreshToken) == 0 {
			return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errInvalidRequest})
		}

		// リフレッシュトークン確認
		token, err := store.GetTokenByRefresh(req.RefreshToken)
		if err != nil {
			switch err {
			case ErrTokenNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

		// クライアント確認
		client, err := store.GetClient(token.ClientID)
		if err != nil {
			switch err {
			case ErrClientNotFound:
				return c.JSON(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		if client.Confidential { // need to authenticate client
			id, pw, err := getCIDAndCPW()
			if err != nil {
				return c.JSON(http.StatusBadRequest, err)
			}
			if client.ID != id || client.Secret != pw {
				return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidClient})
			}
		}

		// 要求スコープ確認
		reqScopes, err := SplitAndValidateScope(req.Scope)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err)
		}
		newScopes := token.GetAvailableScopes(reqScopes)
		if len(reqScopes) == 0 {
			newScopes = token.Scopes
		} else if len(newScopes) == 0 {
			return c.JSON(http.StatusBadRequest, ErrInvalidScope)
		}

		// トークン発行
		newToken, err := IssueAccessToken(client, token.UserID, token.RedirectURI, newScopes, AccessTokenExp, IsRefreshEnabled)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		if err := store.DeleteTokenByRefresh(req.RefreshToken); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		res.AccessToken = newToken.AccessToken
		res.RefreshToken = newToken.RefreshToken
		res.ExpiresIn = newToken.ExpiresIn
		if len(token.Scopes) != len(newToken.Scopes) {
			res.Scope = newToken.Scopes.String()
		}

	default: // ERROR
		return c.JSON(http.StatusBadRequest, errorResponse{ErrorType: errUnsupportedGrantType})
	}

	return c.JSON(http.StatusOK, res)
}

// IssueAccessToken : AccessTokenを発行します
func IssueAccessToken(client *Client, userID uuid.UUID, redirectURI string, scope scope.AccessScopes, expire int, refresh bool) (*Token, error) {
	newToken := &Token{
		ID:          uuid.NewV4().String(),
		ClientID:    client.ID,
		UserID:      userID,
		RedirectURI: redirectURI,
		AccessToken: generateRandomString(),
		CreatedAt:   time.Now(),
		ExpiresIn:   expire,
		Scopes:      scope,
	}

	if refresh {
		newToken.RefreshToken = generateRandomString()
	}

	if err := store.SaveToken(newToken); err != nil {
		return nil, err
	}

	return newToken, nil
}
