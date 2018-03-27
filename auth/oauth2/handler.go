package oauth2

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
	"github.com/traPtitech/traQ/auth/openid"
	"github.com/traPtitech/traQ/model"
	"net/http"
	"net/url"
	"strings"
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

	oauth2ContextSession = "oauth2_context"

	// AuthScheme : Authorizationヘッダーのスキーム
	AuthScheme = "Bearer"
)

// Handler OAuth2のハンドラ
type Handler struct {
	Store

	//AccessTokenExp アクセストークンの有効時間(秒)
	AccessTokenExp int
	//AuthorizationCodeExp 認可コードの有効時間(秒)
	AuthorizationCodeExp int
	//IsRefreshEnabled リフレッシュトークンを発行するかどうか
	IsRefreshEnabled bool

	//UserAuthenticator ユーザー認証を行う関数
	UserAuthenticator func(id, pw string) (uuid.UUID, error)
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

type authorizeRequest struct {
	ResponseType string `query:"response_type" form:"response_type"`
	ClientID     string `query:"client_id"     form:"client_id"`
	RedirectURI  string `query:"redirect_uri"  form:"redirect_uri"`
	RawScope     string `query:"scope"         form:"scope"`
	State        string `query:"state"         form:"state"`

	CodeChallenge       string `query:"code_challenge"        form:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method" form:"code_challenge_method"`

	Nonce  string `query:"nonce"  form:"nonce"`
	Prompt string `query:"prompt" form:"prompt"`

	Scopes      scope.AccessScopes
	ValidScopes scope.AccessScopes
	Types       responseType
	AccessTime  time.Time
}

type responseType struct {
	Code    bool
	Token   bool
	IDToken bool
	None    bool
}

func (t responseType) valid() bool {
	if t.None {
		return !t.Code && !t.Token && !t.IDToken
	}
	return t.Code || t.Token || t.IDToken
}

// AuthorizationEndpointHandler : 認可エンドポイントのハンドラ
func (store *Handler) AuthorizationEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := authorizeRequest{}
	if err := c.Bind(&req); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, err) //普通は起こらないはず
	}
	if len(req.ClientID) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	req.AccessTime = time.Now()

	// クライアント確認
	client, err := store.GetClient(req.ClientID)
	if err != nil {
		switch err {
		case ErrClientNotFound:
			return echo.NewHTTPError(http.StatusBadRequest)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if client.RedirectURI == "" {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// リダイレクトURI確認
	if len(req.RedirectURI) > 0 && client.RedirectURI != req.RedirectURI {
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	redirectURI := client.RedirectURI

	q := url.Values{}
	if len(req.State) > 0 {
		q.Set("state", req.State)
	}

	// PKCE確認
	if len(req.CodeChallengeMethod) > 0 {
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
		if !pkceStringValidator.MatchString(req.CodeChallenge) {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
	}

	// スコープ確認
	reqScopes, err := SplitAndValidateScope(req.RawScope)
	if err != nil {
		q.Set("error", errInvalidScope)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}
	req.Scopes = reqScopes
	req.ValidScopes = client.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		req.ValidScopes = client.Scopes
	} else if len(req.ValidScopes) == 0 {
		q.Set("error", errInvalidScope)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}

	// ResponseType確認
	types := responseType{false, false, false, false}
	for _, v := range strings.Fields(req.ResponseType) {
		switch v {
		case "code":
			types.Code = true
		case "token":
			types.Token = true
		case "id_token":
			types.IDToken = true
		case "none":
			types.None = true
		default:
			q.Set("error", errUnsupportedResponseType)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
	}
	if !types.valid() {
		q.Set("error", errUnsupportedResponseType)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}
	req.Types = types

	// セッション確認
	se, err := session.Get("sessions", c)
	if err != nil {
		c.Logger().Error(err)
		q.Set("error", errServerError)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}
	var userID string
	if se.Values["userID"] != nil {
		userID = se.Values["userID"].(string)
	}

	switch req.Prompt {
	case "":
		break

	case "none":
		u, err := model.GetUser(userID)
		if err != nil {
			switch err {
			case model.ErrNotFound:
				q.Set("error", errLoginRequired)
				return c.Redirect(http.StatusFound, redirectURI+q.Encode())
			default:
				c.Logger().Error(err)
				q.Set("error", errServerError)
				return c.Redirect(http.StatusFound, redirectURI+q.Encode())
			}
		}

		tokens, err := store.GetTokensByUser(uuid.FromStringOrNil(u.ID))
		if err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
		ok := false
		for _, v := range tokens {
			if v.ClientID == req.ClientID {
				all := true
				for _, s := range req.Scopes {
					if !v.Scopes.Contains(s) {
						all = false
						break
					}
				}
				if all {
					ok = true
					break
				}
			}
		}
		if !ok {
			q.Set("error", errConsentRequired)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}

		data := &AuthorizeData{
			Code:                generateRandomString(),
			ClientID:            req.ClientID,
			UserID:              uuid.FromStringOrNil(userID),
			CreatedAt:           time.Now(),
			ExpiresIn:           store.AuthorizationCodeExp,
			RedirectURI:         req.RedirectURI,
			Scopes:              req.ValidScopes,
			OriginalScopes:      req.Scopes,
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
			Nonce:               req.Nonce,
		}
		if err := store.SaveAuthorize(data); err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
		q.Set("code", data.Code)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())

	default:
		q.Set("error", errInvalidRequest)
		q.Set("error_description", fmt.Sprintf("prompt %s is not supported", req.Prompt))
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}

	switch {
	case types.Code && !types.Token && !types.IDToken: // "code" 現状はcodeしかサポートしない
		se.Values[oauth2ContextSession] = req
		if err := se.Save(c.Request(), c.Response()); err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}

		q.Set("client_id", req.ClientID)
		q.Set("scopes", req.ValidScopes.String())
		return c.Redirect(http.StatusFound, "/login"+q.Encode())
	}

	q.Set("error", errUnsupportedResponseType)
	return c.Redirect(http.StatusFound, redirectURI+q.Encode())
}

// AuthorizationDecideHandler : 認可エンドポイントの確認フォームのハンドラ
func (store *Handler) AuthorizationDecideHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")
	userID := c.Get("user").(*model.User).ID

	req := struct {
		Submit string `form:"submit"`
	}{}
	if err := c.Bind(&req); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, err) //普通は起こらないはず
	}

	// セッション確認
	se, err := session.Get("sessions", c)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	reqAuth, ok := se.Values[oauth2ContextSession].(authorizeRequest)
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	se.Values[oauth2ContextSession] = nil
	if err := se.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// クライアント確認
	client, err := store.GetClient(reqAuth.ClientID)
	if err != nil {
		switch err {
		case ErrClientNotFound:
			return echo.NewHTTPError(http.StatusBadRequest)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if client.RedirectURI == "" { // RedirectURIが事前登録されていない
		return echo.NewHTTPError(http.StatusForbidden)
	}
	redirectURI := client.RedirectURI

	q := url.Values{}
	if len(reqAuth.State) > 0 {
		q.Set("state", reqAuth.State)
	}

	// タイムアウト
	if reqAuth.AccessTime.Add(5 * time.Minute).After(time.Now()) {
		q.Set("error", errAccessDenied)
		q.Set("error_description", "timeout")
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}
	// 拒否
	if req.Submit != "approve" {
		q.Set("error", errAccessDenied)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}

	switch {
	case reqAuth.Types.Code && !reqAuth.Types.Token && !reqAuth.Types.IDToken: // "code" 現状はcodeしかサポートしない
		data := &AuthorizeData{
			Code:                generateRandomString(),
			ClientID:            reqAuth.ClientID,
			UserID:              uuid.FromStringOrNil(userID),
			CreatedAt:           time.Now(),
			ExpiresIn:           store.AuthorizationCodeExp,
			RedirectURI:         reqAuth.RedirectURI,
			Scopes:              reqAuth.ValidScopes,
			OriginalScopes:      reqAuth.Scopes,
			CodeChallenge:       reqAuth.CodeChallenge,
			CodeChallengeMethod: reqAuth.CodeChallengeMethod,
			Nonce:               reqAuth.Nonce,
		}
		if err := store.SaveAuthorize(data); err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			return c.Redirect(http.StatusFound, redirectURI+q.Encode())
		}
		q.Set("code", data.Code)

	default:
		q.Set("error", errUnsupportedResponseType)
	}

	return c.Redirect(http.StatusFound, redirectURI+q.Encode())
}

// TokenEndpointHandler : トークンエンドポイントのハンドラ
func (store *Handler) TokenEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := tokenRequest{}
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
			pw = req.ClientSecret
		}
		return
	}
	res := &tokenResponse{
		TokenType: AuthScheme,
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
		newToken, err := store.IssueAccessToken(client, code.UserID, client.RedirectURI, code.Scopes, store.AccessTokenExp, store.IsRefreshEnabled)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		// OpenID Connect IDToken発行
		if code.Scopes.Contains(scope.OpenID) && openid.Available() {
			idToken := openid.NewIDToken(newToken.CreatedAt, int64(store.AccessTokenExp))
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
		uid, err := store.UserAuthenticator(req.Username, req.Password)
		if err != nil {
			switch err {
			case ErrUserIDOrPasswordWrong:
				return c.JSON(http.StatusUnauthorized, errorResponse{ErrorType: errInvalidGrant})
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
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
		newToken, err := store.IssueAccessToken(client, uid, client.RedirectURI, validScopes, store.AccessTokenExp, store.IsRefreshEnabled)
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
		newToken, err := store.IssueAccessToken(client, uuid.Nil, client.RedirectURI, validScopes, store.AccessTokenExp, false)
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
		newToken, err := store.IssueAccessToken(client, token.UserID, token.RedirectURI, newScopes, store.AccessTokenExp, store.IsRefreshEnabled)
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
func (store *Handler) IssueAccessToken(client *Client, userID uuid.UUID, redirectURI string, scope scope.AccessScopes, expire int, refresh bool) (*Token, error) {
	newToken := &Token{
		ID:          uuid.NewV4(),
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
