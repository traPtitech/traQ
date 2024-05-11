package oauth2

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/go-querystring/query"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
)

func init() {
	gob.Register(authorizeRequest{})
}

type authorizeRequest struct {
	ResponseType string `query:"response_type" form:"response_type" url:"response_type,omitempty"`
	ClientID     string `query:"client_id"     form:"client_id"     url:"client_id"`
	RedirectURI  string `query:"redirect_uri"  form:"redirect_uri"  url:"redirect_uri,omitempty"`
	RawScope     string `query:"scope"         form:"scope"         url:"scope,omitempty"`
	State        string `query:"state"         form:"state"         url:"state,omitempty"`

	CodeChallenge       string `query:"code_challenge"        form:"code_challenge"        url:"code_challenge,omitempty"`
	CodeChallengeMethod string `query:"code_challenge_method" form:"code_challenge_method" url:"code_challenge_method,omitempty"`

	Nonce  string `query:"nonce"  form:"nonce"  url:"nonce,omitempty"`
	Prompt string `query:"prompt" form:"prompt" url:"prompt,omitempty"`

	Scopes      model.AccessScopes `url:"-"`
	ValidScopes model.AccessScopes `url:"-"`
	Types       responseType       `url:"-"`
	AccessTime  time.Time          `url:"-"`
}

func (r authorizeRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ClientID, vd.Required),
	)
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

// AuthorizationEndpointHandler 認可エンドポイントのハンドラ
func (h *Handler) AuthorizationEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	var req authorizeRequest
	if err := extension.BindAndValidate(c, &req); err != nil {
		return err
	}
	req.AccessTime = time.Now()

	// クライアント確認
	client, err := h.Repo.GetClient(req.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.BadRequest("unknown client")
		default:
			return herror.InternalServerError(err)
		}
	}
	if len(client.RedirectURI) == 0 {
		return herror.Forbidden("invalid client")
	}

	// リダイレクトURI確認
	if len(req.RedirectURI) > 0 && client.RedirectURI != req.RedirectURI {
		return herror.BadRequest("invalid client")
	}
	redirectURI, _ := url.ParseRequestURI(client.RedirectURI)

	q := &url.Values{}
	if len(req.State) > 0 {
		q.Set("state", req.State)
	}

	// PKCE確認
	if len(req.CodeChallengeMethod) > 0 {
		if !lo.Contains(supportedCodeChallengeMethods, req.CodeChallengeMethod) {
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
	var types responseType
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
	se, err := h.SessStore.GetSession(c)
	if err != nil && err != session.ErrSessionNotFound {
		h.L(c).Error(err.Error(), zap.Error(err))
		q.Set("error", errServerError)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}

	switch req.Prompt {
	case "":
		break

	case "none":
		if se == nil {
			q.Set("error", errLoginRequired)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
		u, err := h.Repo.GetUser(se.UserID(), false)
		if err != nil {
			h.L(c).Error(err.Error(), zap.Error(err))
			q.Set("error", errServerError)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}

		tokens, err := h.Repo.GetTokensByUser(u.GetID())
		if err != nil {
			h.L(c).Error(err.Error(), zap.Error(err))
			q.Set("error", errServerError)
			redirectURI.RawQuery = q.Encode()
			return c.Redirect(http.StatusFound, redirectURI.String())
		}
		ok := false
		for _, v := range tokens {
			if v.ClientID == req.ClientID {
				all := true
				for s := range req.Scopes {
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
			Code:                random.SecureAlphaNumeric(36),
			ClientID:            req.ClientID,
			UserID:              se.UserID(),
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
			h.L(c).Error(err.Error(), zap.Error(err))
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
	case types.Code && !types.Token && !types.IDToken: // 現状は Authorization Code Flow しかサポートしない
		if se == nil {
			// 未ログインの場合はログインしてから再度叩かせる
			current := c.Request().URL
			v, _ := query.Values(req)
			current.RawQuery = v.Encode() // POSTの場合を考慮して再エンコード

			var loginURL url.URL
			loginURL.Path = "/login"
			loginQuery := &url.Values{}
			loginQuery.Set("redirect", current.String())
			loginURL.RawQuery = loginQuery.Encode()
			return c.Redirect(http.StatusFound, loginURL.String())
		}

		if err := se.Set(oauth2ContextSession, req); err != nil {
			h.L(c).Error(err.Error(), zap.Error(err))
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

type authorizationDecideHandlerRequest struct {
	Submit string `form:"submit"`
}

func (r authorizationDecideHandlerRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Submit, vd.Required),
	)
}

// AuthorizationDecideHandler 認可エンドポイントの確認フォームのハンドラ
func (h *Handler) AuthorizationDecideHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	var req authorizationDecideHandlerRequest
	if err := extension.BindAndValidate(c, &req); err != nil {
		return err
	}

	// セッション確認
	se, err := h.SessStore.GetSession(c)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if se == nil {
		return herror.Forbidden("bad session")
	}

	_reqAuth, err := se.Get(oauth2ContextSession)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if _reqAuth == nil {
		return herror.Forbidden("bad session")
	}
	reqAuth := _reqAuth.(authorizeRequest)
	if err := se.Delete(oauth2ContextSession); err != nil {
		return herror.InternalServerError(err)
	}

	// クライアント確認
	client, err := h.Repo.GetClient(reqAuth.ClientID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.BadRequest("unknown client")
		default:
			return herror.InternalServerError(err)
		}
	}
	if client.RedirectURI == "" { // RedirectURIが事前登録されていない
		return herror.Forbidden("invalid client")
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
	case reqAuth.Types.Code && !reqAuth.Types.Token && !reqAuth.Types.IDToken: // 現状は Authorization Code Flow しかサポートしない
		data := &model.OAuth2Authorize{
			Code:                random.SecureAlphaNumeric(36),
			ClientID:            reqAuth.ClientID,
			UserID:              se.UserID(),
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
			h.L(c).Error(err.Error(), zap.Error(err))
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
