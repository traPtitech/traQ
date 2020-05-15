package oauth2

import (
	"encoding/gob"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/random"
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

	Scopes      model.AccessScopes
	ValidScopes model.AccessScopes
	Types       responseType
	AccessTime  time.Time
}

func (r authorizeRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ClientID, vd.Required),
	)
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
		h.L(c).Error(err.Error(), zap.Error(err))
		q.Set("error", errServerError)
		redirectURI.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, redirectURI.String())
	}
	userID := se.GetUserID()

	switch req.Prompt {
	case "":
		break

	case "none":
		u, err := h.Repo.GetUser(userID, false)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				q.Set("error", errLoginRequired)
			default:
				h.L(c).Error(err.Error(), zap.Error(err))
				q.Set("error", errServerError)
			}
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
	case types.Code && !types.Token: // "code" 現状はcodeしかサポートしない
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
	se, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if se == nil {
		return herror.Forbidden("bad session")
	}

	reqAuth, ok := se.Get(oauth2ContextSession).(authorizeRequest)
	if !ok {
		return herror.Forbidden("bad session")
	}
	userID := se.GetUserID()
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
	case reqAuth.Types.Code && !reqAuth.Types.Token: // "code" 現状はcodeしかサポートしない
		data := &model.OAuth2Authorize{
			Code:                random.SecureAlphaNumeric(36),
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
