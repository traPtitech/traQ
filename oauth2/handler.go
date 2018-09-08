package oauth2

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2/scope"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	gob.Register(authorizeRequest{})
}

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

	// AuthScheme Authorizationヘッダーのスキーム
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
	//UserInfoGetter ユーザー情報を取得する関数
	UserInfoGetter func(uid uuid.UUID) (UserInfo, error)

	// OpenID Connect用
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	Issuer     string
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
		return echo.NewHTTPError(http.StatusBadRequest, err)
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
	redirectURI, _ := url.ParseRequestURI(client.RedirectURI)

	q := &url.Values{}
	if len(req.State) > 0 {
		q.Set("state", req.State)
	}

	// PKCE確認
	if len(req.CodeChallengeMethod) > 0 {
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			q.Set("error", errInvalidRequest)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
		if !pkceStringValidator.MatchString(req.CodeChallenge) {
			q.Set("error", errInvalidRequest)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
	}

	// スコープ確認
	reqScopes, err := SplitAndValidateScope(req.RawScope)
	if err != nil {
		q.Set("error", errInvalidScope)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	req.Scopes = reqScopes
	req.ValidScopes = client.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		req.ValidScopes = client.Scopes
	} else if len(req.ValidScopes) == 0 {
		q.Set("error", errInvalidScope)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
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
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
	}
	if !types.valid() {
		q.Set("error", errUnsupportedResponseType)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	req.Types = types

	// セッション確認
	se, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		c.Logger().Error(err)
		q.Set("error", errServerError)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	userID := se.GetUserID()

	switch req.Prompt {
	case "":
		break

	case "none":
		u, err := store.UserInfoGetter(userID)
		if err != nil {
			switch err {
			case ErrUserIDOrPasswordWrong:
				q.Set("error", errLoginRequired)
				redirectURI.RawQuery = q.Encode()
				return c.Redirect(http.StatusFound, redirectURI.String())
			default:
				c.Logger().Error(err)
				q.Set("error", errServerError)
				redirectURI.RawQuery = q.Encode()
				return c.Redirect(http.StatusFound, redirectURI.String())
			}
		}

		tokens, err := store.GetTokensByUser(u.GetUID())
		if err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
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
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}

		data := &AuthorizeData{
			Code:                generateRandomString(),
			ClientID:            req.ClientID,
			UserID:              userID,
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
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
		q.Set("code", data.Code)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())

	default:
		q.Set("error", errInvalidRequest)
		q.Set("error_description", fmt.Sprintf("prompt %s is not supported", req.Prompt))
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}

	switch {
	case types.Code && !types.Token && !types.IDToken: // "code" 現状はcodeしかサポートしない
		if err := se.Set(oauth2ContextSession, req); err != nil {
			c.Logger().Error(err)
			q.Set("error", errServerError)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}

		q.Set("client_id", req.ClientID)
		q.Set("scopes", req.ValidScopes.String())
		return c.Redirect(http.StatusFound, "/login?"+q.Encode())
	}

	q.Set("error", errUnsupportedResponseType)
	redirectURI.RawQuery = q.Encode()
	return c.Redirect(http.StatusFound, redirectURI.String())
}

// AuthorizationDecideHandler : 認可エンドポイントの確認フォームのハンドラ
func (store *Handler) AuthorizationDecideHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := struct {
		Submit string `form:"submit"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// セッション確認
	se, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if se == nil {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	reqAuth, ok := se.Get(oauth2ContextSession).(authorizeRequest)
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	userID := se.GetUserID()
	if err := se.Delete(oauth2ContextSession); err != nil {
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
	redirectURI, _ := url.ParseRequestURI(client.RedirectURI)

	q := url.Values{}
	if len(reqAuth.State) > 0 {
		q.Set("state", reqAuth.State)
	}

	// タイムアウト
	if reqAuth.AccessTime.Add(5 * time.Minute).Before(time.Now()) {
		q.Set("error", errAccessDenied)
		q.Set("error_description", "timeout")
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	// 拒否
	if req.Submit != "approve" {
		q.Set("error", errAccessDenied)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}

	switch {
	case reqAuth.Types.Code && !reqAuth.Types.Token && !reqAuth.Types.IDToken: // "code" 現状はcodeしかサポートしない
		data := &AuthorizeData{
			Code:                generateRandomString(),
			ClientID:            reqAuth.ClientID,
			UserID:              userID,
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
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
		q.Set("code", data.Code)

	default:
		q.Set("error", errUnsupportedResponseType)
	}

	redirectURI.RawQuery = q.Encode()
	return c.Redirect(http.StatusFound, redirectURI.String())
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
		if code.Scopes.Contains(scope.OpenID) && store.IsOpenIDConnectAvailable() {
			idToken := store.NewIDToken(newToken.CreatedAt, int64(store.AccessTokenExp))
			idToken.Audience = code.ClientID
			idToken.Subject = code.UserID.String()
			idToken.Nonce = code.Nonce

			user, err := store.UserInfoGetter(code.UserID)
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			if code.Scopes.Contains(scope.Profile) {
				idToken.Name = user.GetName()
			}

			res.IDToken, err = idToken.Generate(store.privateKey)
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
		UserID:      userID,
		RedirectURI: redirectURI,
		AccessToken: generateRandomString(),
		CreatedAt:   time.Now(),
		ExpiresIn:   expire,
		Scopes:      scope,
	}

	if client != nil {
		newToken.ClientID = client.ID
	}

	if refresh {
		newToken.RefreshToken = generateRandomString()
	}

	if err := store.SaveToken(newToken); err != nil {
		return nil, err
	}

	return newToken, nil
}

// LoadKeys OpenID Connectのjwt用のRSA秘密鍵・公開鍵を読み込みます
func (store *Handler) LoadKeys(private, public []byte) (err error) {
	store.privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		return err
	}
	store.publicKey, err = jwt.ParseRSAPublicKeyFromPEM(public)
	return
}

// IsOpenIDConnectAvailable OpenID Connectが有効かどうかを返します
func (store *Handler) IsOpenIDConnectAvailable() bool {
	return store.privateKey != nil && store.publicKey != nil && store.privateKey.Validate() == nil
}

// NewIDToken IDTokenを生成します
func (store *Handler) NewIDToken(issueAt time.Time, expireIn int64) *IDToken {
	return &IDToken{
		StandardClaims: jwt.StandardClaims{
			Issuer:    store.Issuer,
			IssuedAt:  issueAt.Unix(),
			ExpiresAt: issueAt.Unix() + expireIn,
		},
	}
}

// PublicKeysHandler publishes the public signing keys.
func (store *Handler) PublicKeysHandler(c echo.Context) error {
	if store.IsOpenIDConnectAvailable() {
		data := make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(store.publicKey.E))

		res := map[string]interface{}{
			"keys": map[string]interface{}{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(store.publicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bytes.TrimLeft(data, "\x00")),
			},
		}

		return c.JSON(http.StatusOK, res)
	}

	return echo.NewHTTPError(http.StatusNotFound)
}

// DiscoveryHandler returns the OpenID Connect discovery object.
func (store *Handler) DiscoveryHandler(c echo.Context) error {
	if store.IsOpenIDConnectAvailable() {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"issuer":                                store.Issuer,
			"authorization_endpoint":                store.Issuer + "/api/1.0/oauth2/authorize",
			"token_endpoint":                        store.Issuer + "/api/1.0/oauth2/token",
			"jwks_uri":                              store.Issuer + "/publickeys",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"scopes_supported":                      []string{"openid", "profile"},
			"grantTypesSupported":                   []string{"authorization_code", "refresh_token", "client_credentials", "password"},
			"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
			"display_values_supported":              []string{"page"},
			"ui_locales_supported":                  []string{"ja"},
			"request_parameter_supported":           false,
			"request_uri_parameter_supported":       false,
			"claims_supported": []string{
				"aud", "exp", "iat", "iss", "name", "sub",
			},
		})
	}
	return echo.NewHTTPError(http.StatusNotFound)
}
