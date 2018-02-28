package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var pkceStringValidator = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")

type authorizeRequest struct {
	ResponseType string `query:"response_type"`
	ClientID     string `query:"client_id"`
	RedirectURI  string `query:"redirect_uri"`
	Scope        string `query:"scope"`
	State        string `query:"state"`

	CodeChallenge       string `query:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method"`

	Nonce  string `query:"nonce"`
	Prompt string `query:"prompt"`
}

type authorizeRequestPost struct {
	ResponseType string `form:"response_type"`
	ClientID     string `form:"client_id"`
	RedirectURI  string `form:"redirect_uri"`
	Scope        string `form:"scope"`
	State        string `form:"state"`

	CodeChallenge       string `form:"code_challenge"`
	CodeChallengeMethod string `form:"code_challenge_method"`

	Submit string `form:"submit"`
}

type responseType struct {
	Code    bool
	Token   bool
	IDToken bool
}

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
	req := &authorizeRequest{}
	if err := c.Bind(&req); err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, err) //普通は起こらないはず
	}

	if len(req.ClientID) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	client, err := store.GetClient(req.ClientID)
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if client == nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if client.RedirectURI == "" {
		return echo.NewHTTPError(http.StatusForbidden)
	} else if client.RedirectURI != req.RedirectURI {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	q := url.Values{}
	if len(req.State) > 0 {
		q.Set("state", req.State)
	}

	if len(req.CodeChallengeMethod) > 0 {
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
		if !pkceStringValidator.MatchString(req.CodeChallenge) {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
	}

	reqScopes, err := splitAndValidateScope(req.Scope)
	if err != nil {
		q.Set("error", errInvalidScope)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}

	var (
		validScopes scope.AccessScopes
		userID      string
	)
	if len(reqScopes) > 0 {
		for _, s := range reqScopes {
			if client.Scope.Contains(s) {
				validScopes = append(validScopes, s)
			}
		}
	} else {
		validScopes = client.Scope
	}

	types := responseType{false, false, false}
	for _, v := range strings.Split(req.ResponseType, " ") {
		if v == "code" {
			types.Code = true
		} else if v == "token" {
			types.Token = true
		} else if v == "id_token" {
			types.IDToken = true
		} else {
			q.Set("error", errUnsupportedResponseType)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
	}

	switch {
	case types.Code && !types.Token && !types.IDToken: // "code"
		return c.Redirect(http.StatusFound, "") //FIXME 認可ページに飛ばす
	default:
		q.Set("error", errUnsupportedResponseType)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}
}

func PostAuthorizationEndpointHandler(c echo.Context) error {
	req := &authorizeRequestPost{}
	if err := c.Bind(&req); err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, err) //普通は起こらないはず
	}

	if len(req.ClientID) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	client, err := store.GetClient(req.ClientID)
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if client == nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if client.RedirectURI == "" {
		return echo.NewHTTPError(http.StatusForbidden)
	} else if client.RedirectURI != req.RedirectURI {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	q := url.Values{}
	if len(req.State) > 0 {
		q.Set("state", req.State)
	}

	if len(req.CodeChallengeMethod) > 0 {
		if req.CodeChallengeMethod != "plain" && req.CodeChallengeMethod != "S256" {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
		if !pkceStringValidator.MatchString(req.CodeChallenge) {
			q.Set("error", errInvalidRequest)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
	}

	reqScopes, err := splitAndValidateScope(req.Scope)
	if err != nil {
		q.Set("error", errInvalidScope)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}

	var (
		validScopes scope.AccessScopes
		userID      string
	)
	if len(reqScopes) > 0 {
		for _, s := range reqScopes {
			if client.Scope.Contains(s) {
				validScopes = append(validScopes, s)
			}
		}
	} else {
		validScopes = client.Scope
	}

	types := responseType{false, false, false}
	for _, v := range strings.Split(req.ResponseType, " ") {
		if v == "code" {
			types.Code = true
		} else if v == "token" {
			types.Token = true
		} else if v == "id_token" {
			types.IDToken = true
		} else {
			q.Set("error", errUnsupportedResponseType)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
	}

	se, err := session.Get("sessions", c)
	if err != nil {
		c.Echo().Logger.Errorf("Failed to get a session: %v", err)
		q.Set("error", errServerError)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}
	if se.Values["userID"] != nil {
		userID = se.Values["userID"].(string)
	}
	if userID == "" { //未ログイン
		q.Set("error", errAccessDenied)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}

	if req.Submit != "approve" { //拒否
		q.Set("error", errAccessDenied)
		return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
	}

	switch {
	case types.Code && !types.Token && !types.IDToken: // "code"
		data := &AuthorizeData{
			Code:                generateRandomString(),
			ClientID:            client.ID,
			UserID:              uuid.FromStringOrNil(userID),
			CreatedAt:           time.Now(),
			ExpiresIn:           AuthorizationCodeExp,
			RedirectURI:         client.RedirectURI,
			Scope:               validScopes,
			OriginalScope:       reqScopes,
			CodeChallenge:       req.CodeChallenge,
			CodeChallengeMethod: req.CodeChallengeMethod,
		}
		if err := store.SaveAuthorize(data); err != nil {
			c.Echo().Logger.Error(err)
			q.Set("error", errServerError)
			return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
		}
		q.Set("code", data.Code)

	default:
		q.Set("error", errUnsupportedResponseType)
	}

	return c.Redirect(http.StatusFound, client.RedirectURI+q.Encode())
}
