package oauth2

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
	"net/http"
	"time"
)

// Token : OAuth2.0 Access Token構造体
type Token struct {
	ClientID     string
	UserID       uuid.UUID
	RedirectURI  string
	AccessToken  string
	RefreshToken string
	CreatedAt    time.Time
	ExpiresIn    int
	Scope        scope.AccessScopes
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
}

// TokenEndpointHandler : トークンエンドポイントのハンドラ
func TokenEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := &tokenRequest{}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidRequest})
	}

	res := &tokenResponse{}
	switch req.GrantType {
	case "authorization_code": // Authorization Code Grant

	case "password": // Resource Owner Password Credentials Grant

	case "client_credentials": // Client Credentials Grant

	case "refresh_token": // Refreshing an Access Token
		if len(req.RefreshToken) == 0 {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidRequest})
		}

		token, err := store.GetTokenByRefresh(req.RefreshToken)
		if err != nil {
			c.Echo().Logger.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if token == nil {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidGrant})
		}

		client, err := store.GetClient(token.ClientID)
		if err != nil {
			c.Echo().Logger.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if client == nil {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidClient})
		}

		if client.Confidential { // need to authenticate client
			id, pw, ok := c.Request().BasicAuth()
			if !ok { // Request Body
				if len(req.ClientID) == 0 {
					return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidClient})
				}
				id = req.ClientID
				pw = req.Password
			}
			if client.ID != id {
				return c.JSON(http.StatusUnauthorized, errorResponse{Error: errInvalidClient})
			}
			if client.Secret != pw {
				return c.JSON(http.StatusUnauthorized, errorResponse{Error: errInvalidClient})
			}
		}

		reqScopes, err := splitAndValidateScope(req.Scope)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: errInvalidScope})
		}

		var newScopes scope.AccessScopes
		if len(reqScopes) > 0 {
			for _, req := range reqScopes {
				if token.Scope.Contains(req) {
					newScopes = append(newScopes, req)
				}
			}
		} else {
			newScopes = nil
		}

		newToken := &Token{
			ClientID:    client.ID,
			UserID:      token.UserID,
			RedirectURI: token.RedirectURI,
			AccessToken: generateRandomString(),
			CreatedAt:   time.Now(),
			ExpiresIn:   AccessTokenExp,
			Scope:       newScopes,
		}

		if IsRefreshEnabled {
			newToken.RefreshToken = generateRandomString()
		}

		if err := store.DeleteTokenByRefresh(req.RefreshToken); err != nil {
			c.Echo().Logger.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		if err := store.SaveToken(newToken); err != nil { // get validity
			c.Echo().Logger.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		res.TokenType = "Bearer"
		res.AccessToken = newToken.AccessToken
		res.RefreshToken = newToken.RefreshToken
		res.Scope = newToken.Scope.String()
		res.ExpiresIn = newToken.ExpiresIn

	default: // ERROR
		return c.JSON(http.StatusBadRequest, errorResponse{Error: errUnsupportedGrantType})
	}

	return c.JSON(http.StatusOK, res)
}
