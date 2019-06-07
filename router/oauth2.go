package router

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	gob.Register(authorizeRequest{})
}

const (
	grantTypeAuthorizationCode = "authorization_code"
	grantTypePassword          = "password"
	grantTypeClientCredentials = "client_credentials"
	grantTypeRefreshToken      = "refresh_token"

	errInvalidRequest          = "invalid_request"
	errUnauthorizedClient      = "unauthorized_client"
	errAccessDenied            = "access_denied"
	errUnsupportedResponseType = "unsupported_response_type"
	errInvalidScope            = "invalid_scope"
	errServerError             = "server_error"
	errInvalidClient           = "invalid_client"
	errInvalidGrant            = "invalid_grant"
	errUnsupportedGrantType    = "unsupported_grant_type"
	errLoginRequired           = "login_required"
	errConsentRequired         = "consent_required"

	oauth2ContextSession = "oauth2_context"
	authScheme           = "Bearer"

	authorizationCodeExp = 60 * 5
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type authorizeRequest struct {
	ResponseType string `query:"response_type" form:"response_type"`
	ClientID     string `query:"client_id"     form:"client_id"     validate:"required"`
	RedirectURI  string `query:"redirect_uri"  form:"redirect_uri"`
	RawScope     string `query:"scope"         form:"scope"`
	State        string `query:"state"         form:"state"`

	CodeChallenge       string `query:"code_challenge"        form:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method" form:"code_challenge_method"`

	Nonce  string `query:"nonce"  form:"nonce"`
	Prompt string `query:"prompt" form:"prompt"`

	Scopes      model.AccessScopes
	ValidScopes model.AccessScopes
	Types       responseType
	AccessTime  time.Time
}

type responseType struct {
	Code  bool
	Token bool
	None  bool
}

func (t responseType) valid() bool {
	if t.None {
		return !t.Code && !t.Token
	}
	return t.Code || t.Token
}

type oauth2ErrorResponse struct {
	ErrorType        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// AuthorizationEndpointHandler 認可エンドポイントのハンドラ
func (h *Handlers) AuthorizationEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := authorizeRequest{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}
	req.AccessTime = time.Now()

	// クライアント確認
	client, err := h.Repo.GetClient(req.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return badRequest("unknown client")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	if len(client.RedirectURI) == 0 {
		return forbidden("invalid client")
	}

	// リダイレクトURI確認
	if len(req.RedirectURI) > 0 && client.RedirectURI != req.RedirectURI {
		return badRequest("invalid client")
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
		if !validator.PKCERegex.MatchString(req.CodeChallenge) {
			q.Set("error", errInvalidRequest)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
	}

	// スコープ確認
	reqScopes, err := h.splitAndValidateScope(req.RawScope)
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
	types := responseType{false, false, false}
	for _, v := range strings.Fields(req.ResponseType) {
		switch v {
		case "code":
			types.Code = true
		case "token":
			types.Token = true
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		q.Set("error", errServerError)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	userID := se.GetUserID()

	switch req.Prompt {
	case "":
		break

	case "none":
		u, err := h.Repo.GetUser(userID)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				q.Set("error", errLoginRequired)
			default:
				h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
				q.Set("error", errServerError)
			}
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}

		tokens, err := h.Repo.GetTokensByUser(u.ID)
		if err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
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

		data := &model.OAuth2Authorize{
			Code:                utils.RandAlphabetAndNumberString(36),
			ClientID:            req.ClientID,
			UserID:              userID,
			CreatedAt:           time.Now(),
			ExpiresIn:           authorizationCodeExp,
			RedirectURI:         req.RedirectURI,
			Scopes:              req.ValidScopes,
			OriginalScopes:      req.Scopes,
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
			Nonce:               req.Nonce,
		}
		if err := h.Repo.SaveAuthorize(data); err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
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
	case types.Code && !types.Token: // "code" 現状はcodeしかサポートしない
		if err := se.Set(oauth2ContextSession, req); err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			q.Set("error", errServerError)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}

		q.Set("client_id", req.ClientID)
		q.Set("scopes", req.ValidScopes.String())
		return c.Redirect(http.StatusFound, "/consent?"+q.Encode())
	default:
		q.Set("error", errUnsupportedResponseType)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
}

// AuthorizationDecideHandler 認可エンドポイントの確認フォームのハンドラ
func (h *Handlers) AuthorizationDecideHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := struct {
		Submit string `form:"submit"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// セッション確認
	se, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	if se == nil {
		return forbidden("bad session")
	}

	reqAuth, ok := se.Get(oauth2ContextSession).(authorizeRequest)
	if !ok {
		return forbidden("bad session")
	}
	userID := se.GetUserID()
	if err := se.Delete(oauth2ContextSession); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	// クライアント確認
	client, err := h.Repo.GetClient(reqAuth.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return badRequest("unknown client")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	if client.RedirectURI == "" { // RedirectURIが事前登録されていない
		return forbidden("invalid client")
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
	case reqAuth.Types.Code && !reqAuth.Types.Token: // "code" 現状はcodeしかサポートしない
		data := &model.OAuth2Authorize{
			Code:                utils.RandAlphabetAndNumberString(36),
			ClientID:            reqAuth.ClientID,
			UserID:              userID,
			CreatedAt:           time.Now(),
			ExpiresIn:           authorizationCodeExp,
			RedirectURI:         reqAuth.RedirectURI,
			Scopes:              reqAuth.ValidScopes,
			OriginalScopes:      reqAuth.Scopes,
			CodeChallenge:       reqAuth.CodeChallenge,
			CodeChallengeMethod: reqAuth.CodeChallengeMethod,
			Nonce:               reqAuth.Nonce,
		}
		if err := h.Repo.SaveAuthorize(data); err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
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

// TokenEndpointHandler トークンエンドポイントのハンドラ
func (h *Handlers) TokenEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	switch c.FormValue("grant_type") {
	case grantTypeAuthorizationCode:
		return h.tokenEndpointAuthorizationCodeHandler(c)
	case grantTypePassword:
		return h.tokenEndpointPasswordHandler(c)
	case grantTypeClientCredentials:
		return h.tokenEndpointClientCredentialsHandler(c)
	case grantTypeRefreshToken:
		return h.tokenEndpointRefreshTokenHandler(c)
	default:
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errUnsupportedGrantType})
	}
}

func (h *Handlers) tokenEndpointAuthorizationCodeHandler(c echo.Context) error {
	var req struct {
		Code         string `form:"code" validate:"required"`
		RedirectURI  string `form:"redirect_uri"`
		ClientID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
		CodeVerifier string `form:"code_verifier"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	// 認可コード確認
	code, err := h.Repo.GetAuthorize(req.Code)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	// 認可コードは２回使えない
	if err := h.Repo.DeleteAuthorize(code.Code); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	if code.IsExpired() {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidGrant})
	}

	// クライアント確認
	client, err := h.Repo.GetClient(code.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	id, pw, ok := c.Request().BasicAuth()
	if !ok { // Request Body
		if len(req.ClientID) == 0 {
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		}
		id = req.ClientID
		pw = req.ClientSecret
	}
	if client.ID != id || (client.Confidential && client.Secret != pw) {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidClient})
	}

	// リダイレクトURI確認
	if (len(code.RedirectURI) > 0 && client.RedirectURI != req.RedirectURI) || (len(code.RedirectURI) == 0 && len(req.RedirectURI) > 0) {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidGrant})
	}

	// PKCE確認
	if ok, _ := code.ValidatePKCE(req.CodeVerifier); !ok {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	// トークン発行
	newToken, err := h.Repo.IssueToken(client, code.UserID, client.RedirectURI, code.Scopes, h.AccessTokenExp, h.IsRefreshEnabled)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}

	res := &tokenResponse{
		TokenType:   authScheme,
		AccessToken: newToken.AccessToken,
		ExpiresIn:   newToken.ExpiresIn,
	}
	if len(code.OriginalScopes) != len(newToken.Scopes) {
		res.Scope = newToken.Scopes.String()
	}
	if newToken.IsRefreshEnabled() {
		res.RefreshToken = newToken.RefreshToken
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handlers) tokenEndpointPasswordHandler(c echo.Context) error {
	var req struct {
		Scope        string `form:"scope"`
		Username     string `form:"username" validate:"required"`
		Password     string `form:"password" validate:"required"`
		ClientID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	cid, cpw, ok := c.Request().BasicAuth()
	if !ok { // Request Body
		if len(req.ClientID) == 0 {
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		}
		cid = req.ClientID
		cpw = req.ClientSecret
	}

	// クライアント確認
	client, err := h.Repo.GetClient(cid)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if client.Confidential && client.Secret != cpw {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidClient})
	}

	// ユーザー確認
	user, err := h.Repo.GetUserByName(req.Username)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if model.AuthenticateUser(user, req.Password) != nil {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidGrant})
	}

	// 要求スコープ確認
	reqScopes, err := h.splitAndValidateScope(req.Scope)
	if err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}
	validScopes := client.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		validScopes = client.Scopes
	} else if len(validScopes) == 0 {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}

	// トークン発行
	newToken, err := h.Repo.IssueToken(client, user.ID, client.RedirectURI, validScopes, h.AccessTokenExp, h.IsRefreshEnabled)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}

	res := &tokenResponse{
		TokenType:   authScheme,
		AccessToken: newToken.AccessToken,
		ExpiresIn:   newToken.ExpiresIn,
	}
	if len(reqScopes) != len(validScopes) {
		res.Scope = newToken.Scopes.String()
	}
	if newToken.IsRefreshEnabled() {
		res.RefreshToken = newToken.RefreshToken
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handlers) tokenEndpointClientCredentialsHandler(c echo.Context) error {
	var req struct {
		Scope        string `form:"scope"`
		ClientID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	id, pw, ok := c.Request().BasicAuth()
	if !ok { // Request Body
		if len(req.ClientID) == 0 {
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		}
		id = req.ClientID
		pw = req.ClientSecret
	}

	// クライアント確認
	client, err := h.Repo.GetClient(id)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if !client.Confidential {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errUnauthorizedClient})
	}
	if client.Secret != pw {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidClient})
	}

	// 要求スコープ確認
	reqScopes, err := h.splitAndValidateScope(req.Scope)
	if err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}
	validScopes := client.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		validScopes = client.Scopes
	} else if len(validScopes) == 0 {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}

	// トークン発行
	newToken, err := h.Repo.IssueToken(client, uuid.Nil, client.RedirectURI, validScopes, h.AccessTokenExp, false)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}

	res := &tokenResponse{
		TokenType:   authScheme,
		AccessToken: newToken.AccessToken,
		ExpiresIn:   newToken.ExpiresIn,
	}
	if len(reqScopes) != len(validScopes) {
		res.Scope = newToken.Scopes.String()
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handlers) tokenEndpointRefreshTokenHandler(c echo.Context) error {
	var req struct {
		Scope        string `form:"scope"`
		RefreshToken string `form:"refresh_token" validate:"required"`
		ClientID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	// リフレッシュトークン確認
	token, err := h.Repo.GetTokenByRefresh(req.RefreshToken)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}

	// クライアント確認
	client, err := h.Repo.GetClient(token.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if client.Confidential { // need to authenticate client
		id, pw, ok := c.Request().BasicAuth()
		if !ok { // Request Body
			if len(req.ClientID) == 0 {
				return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidClient})
			}
			id = req.ClientID
			pw = req.ClientSecret
		}
		if client.ID != id || client.Secret != pw {
			return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidClient})
		}
	}

	// 要求スコープ確認
	reqScopes, err := h.splitAndValidateScope(req.Scope)
	if err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}
	newScopes := token.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		newScopes = token.Scopes
	} else if len(newScopes) == 0 {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidScope})
	}

	// トークン発行
	newToken, err := h.Repo.IssueToken(client, token.UserID, token.RedirectURI, newScopes, h.AccessTokenExp, h.IsRefreshEnabled)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	if err := h.Repo.DeleteTokenByRefresh(req.RefreshToken); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}

	res := &tokenResponse{
		TokenType:   authScheme,
		AccessToken: newToken.AccessToken,
		ExpiresIn:   newToken.ExpiresIn,
	}
	if len(token.Scopes) != len(newToken.Scopes) {
		res.Scope = newToken.Scopes.String()
	}
	if newToken.IsRefreshEnabled() {
		res.RefreshToken = newToken.RefreshToken
	}
	return c.JSON(http.StatusOK, res)
}

// RevokeTokenEndpointHandler トークン無効化エンドポイントのハンドラ
func (h *Handlers) RevokeTokenEndpointHandler(c echo.Context) error {
	var req struct {
		Token string `form:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return badRequest(err)
	}

	if len(req.Token) == 0 {
		return c.NoContent(http.StatusOK)
	}

	if err := h.Repo.DeleteTokenByAccess(req.Token); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	if err := h.Repo.DeleteTokenByRefresh(req.Token); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusOK)
}

// splitAndValidateScope スペース区切りのスコープ文字列を分解し、検証します
func (h *Handlers) splitAndValidateScope(str string) (model.AccessScopes, error) {
	var scopes model.AccessScopes
	set := map[model.AccessScope]bool{}

	for _, v := range strings.Fields(str) {
		s := model.AccessScope(v)
		if ok := set[s]; !h.RBAC.IsOAuth2Scope(string(s)) || ok {
			return nil, errors.New(errInvalidScope)
		}
		scopes = append(scopes, s)
		set[s] = true
	}

	return scopes, nil
}
