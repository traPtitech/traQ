package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
	"github.com/traPtitech/traQ/model"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	pkceStringValidator  = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")
	oauth2ContextSession = "oauth2_context"
)

// AuthorizeData : Authorization Code Grant用の認可データ構造体
type AuthorizeData struct {
	Code                string
	ClientID            string
	UserID              uuid.UUID
	CreatedAt           time.Time
	ExpiresIn           int
	RedirectURI         string
	Scope               scope.AccessScopes
	OriginalScope       scope.AccessScopes
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
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

// IsExpired : 有効期限が切れているかどうか
func (data *AuthorizeData) IsExpired() bool {
	return data.CreatedAt.Add(time.Duration(data.ExpiresIn) * time.Second).Before(time.Now())
}

// ValidatePKCE : PKCEの検証を行う
func (data *AuthorizeData) ValidatePKCE(verifier string) (bool, error) {
	if len(verifier) == 0 {
		if len(data.CodeChallenge) == 0 {
			return true, nil
		}
		return false, nil
	}
	if !pkceStringValidator.MatchString(verifier) {
		return false, nil
	}

	if len(data.CodeChallengeMethod) == 0 {
		data.CodeChallengeMethod = "plain"
	}

	switch data.CodeChallengeMethod {
	case "plain":
		return verifier == data.CodeChallenge, nil
	case "S256":
		hash := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(hash[:]) == data.CodeChallenge, nil
	}

	return false, fmt.Errorf("unknown method: %v", data.CodeChallengeMethod)
}

// AuthorizationEndpointHandler : 認可エンドポイントのハンドラ
func AuthorizationEndpointHandler(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	req := &authorizeRequest{}
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
	reqScopes, err := splitAndValidateScope(req.RawScope)
	if err != nil {
		q.Set("error", errInvalidScope)
		return c.Redirect(http.StatusFound, redirectURI+q.Encode())
	}
	req.Scopes = reqScopes
	req.ValidScopes = client.GetAvailableScopes(reqScopes)
	if len(reqScopes) == 0 {
		req.ValidScopes = client.Scope
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
					if !v.Scope.Contains(s) {
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
			ExpiresIn:           AuthorizationCodeExp,
			RedirectURI:         req.RedirectURI,
			Scope:               req.ValidScopes,
			OriginalScope:       req.Scopes,
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
		se.Values[oauth2ContextSession] = *req
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
func AuthorizationDecideHandler(c echo.Context) error {
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
			ExpiresIn:           AuthorizationCodeExp,
			RedirectURI:         reqAuth.RedirectURI,
			Scope:               reqAuth.ValidScopes,
			OriginalScope:       reqAuth.Scopes,
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
