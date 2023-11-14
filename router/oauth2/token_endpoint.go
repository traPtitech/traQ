package oauth2

import (
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/utils"
	jwt2 "github.com/traPtitech/traQ/utils/jwt"
)

type oauth2ErrorResponse struct {
	ErrorType        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// TokenEndpointHandler トークンエンドポイントのハンドラ
func (h *Handler) TokenEndpointHandler(c echo.Context) error {
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

func (h *Handler) issueIDToken(client *model.OAuth2Client, token *model.OAuth2Token, userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"iss": h.Origin,
		"sub": userID.String(),
		"aud": client.ID,
		"exp": token.Deadline().Unix(),
		"iat": token.CreatedAt.Unix(),
	}
	if token.Scopes.Contains("profile") {
		userInfo, err := h.OIDC.GetUserInfo(userID)
		if err != nil {
			return "", err
		}
		claims = utils.MergeMap(userInfo, claims)
	}
	return jwt2.Sign(claims)
}

func (h *Handler) issueToken(client *model.OAuth2Client, userID uuid.UUID, scopes, originalScopes model.AccessScopes, grantTypeRefreshAllowed bool) (*tokenResponse, error) {
	isOIDC := scopes.Contains("openid")
	// OIDCの場合は、Refresh TokenのScopeの管理（主にoffline_access周り）が面倒なので、一律で発行しないことにする
	refresh := h.IsRefreshEnabled && grantTypeRefreshAllowed && !isOIDC
	token, err := h.Repo.IssueToken(client, userID, client.RedirectURI, scopes, h.AccessTokenExp, refresh)
	if err != nil {
		return nil, err
	}
	res := &tokenResponse{
		TokenType:   authScheme,
		AccessToken: token.AccessToken,
		ExpiresIn:   token.ExpiresIn,
	}
	if len(originalScopes) != len(token.Scopes) {
		res.Scope = token.Scopes.String()
	}
	if token.IsRefreshEnabled() {
		res.RefreshToken = token.RefreshToken
	}
	if scopes.Contains("openid") {
		idToken, err := h.issueIDToken(client, token, userID)
		if err != nil {
			return nil, err
		}
		res.IDToken = idToken
	}
	return res, nil
}

type tokenEndpointAuthorizationCodeHandlerRequest struct {
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
}

func (r tokenEndpointAuthorizationCodeHandlerRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Code, vd.Required),
	)
}

func (h *Handler) tokenEndpointAuthorizationCodeHandler(c echo.Context) error {
	var req tokenEndpointAuthorizationCodeHandlerRequest
	if err := extension.BindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	// 認可コード確認
	code, err := h.Repo.GetAuthorize(req.Code)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.L(c).Error(err.Error(), zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	// 認可コードは２回使えない
	if err := h.Repo.DeleteAuthorize(code.Code); err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
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
			h.L(c).Error(err.Error(), zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	id, pw, ok := c.Request().BasicAuth()
	if !ok { // Request Payload
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
	res, err := h.issueToken(client, code.UserID, code.Scopes, code.OriginalScopes, true)
	if err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	return c.JSON(http.StatusOK, res)
}

type tokenEndpointPasswordHandlerRequest struct {
	Scope        string `form:"scope"`
	Username     string `form:"username"`
	Password     string `form:"password"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
}

func (r tokenEndpointPasswordHandlerRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Username, vd.Required),
		vd.Field(&r.Password, vd.Required),
	)
}

func (h *Handler) tokenEndpointPasswordHandler(c echo.Context) error {
	var req tokenEndpointPasswordHandlerRequest
	if err := extension.BindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	cid, cpw, ok := c.Request().BasicAuth()
	if !ok { // Request Payload
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
			h.L(c).Error(err.Error(), zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if client.Confidential && client.Secret != cpw {
		return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidClient})
	}

	// ユーザー確認
	user, err := h.Repo.GetUserByName(req.Username, false)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusUnauthorized, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.L(c).Error(err.Error(), zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if user.Authenticate(req.Password) != nil {
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
	res, err := h.issueToken(client, user.GetID(), validScopes, reqScopes, true)
	if err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) tokenEndpointClientCredentialsHandler(c echo.Context) error {
	var req struct {
		Scope        string `form:"scope"`
		ClientID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
	}
	if err := extension.BindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	id, pw, ok := c.Request().BasicAuth()
	if !ok { // Request Payload
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
			h.L(c).Error(err.Error(), zap.Error(err))
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
	res, err := h.issueToken(client, uuid.Nil, validScopes, reqScopes, false)
	if err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	return c.JSON(http.StatusOK, res)
}

type tokenEndpointRefreshTokenHandlerRequest struct {
	Scope        string `form:"scope"`
	RefreshToken string `form:"refresh_token"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
}

func (r tokenEndpointRefreshTokenHandlerRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.RefreshToken, vd.Required),
	)
}

func (h *Handler) tokenEndpointRefreshTokenHandler(c echo.Context) error {
	var req tokenEndpointRefreshTokenHandlerRequest
	if err := extension.BindAndValidate(c, &req); err != nil {
		return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidRequest})
	}

	// リフレッシュトークン確認
	token, err := h.Repo.GetTokenByRefresh(req.RefreshToken)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.JSON(http.StatusBadRequest, oauth2ErrorResponse{ErrorType: errInvalidGrant})
		default:
			h.L(c).Error(err.Error(), zap.Error(err))
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
			h.L(c).Error(err.Error(), zap.Error(err))
			return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
		}
	}
	if client.Confidential { // need to authenticate client
		id, pw, ok := c.Request().BasicAuth()
		if !ok { // Request Payload
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
	res, err := h.issueToken(client, token.UserID, newScopes, token.Scopes, true)
	if err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	// 旧リフレッシュトークン削除
	if err := h.Repo.DeleteTokenByRefresh(req.RefreshToken); err != nil {
		h.L(c).Error(err.Error(), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, oauth2ErrorResponse{ErrorType: errServerError})
	}
	return c.JSON(http.StatusOK, res)
}
