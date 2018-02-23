package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/labstack/echo"
	"net/http"
	"regexp"
	"time"
)

var pkceStringValidator = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")

type authorizeRequest struct {
	ResponseType        string `query:"response_type"`
	ClientID            string `query:"client_id"`
	RedirectURI         string `query:"redirect_uri"`
	Scope               string `query:"scope"`
	State               string `query:"state"`
	CodeChallenge       string `query:"code_challenge"`
	CodeChallengeMethod string `query:"code_challenge_method"`
}

// AuthorizeData : Authorization Code Grant用の認可データ構造体
type AuthorizeData struct {
	Code                string
	ClientID            string
	CreatedAt           time.Time
	ExpiresIn           int
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// IsExpired : 有効期限が切れているかどうか
func (data *AuthorizeData) IsExpired() bool {
	return data.CreatedAt.Add(time.Duration(data.ExpiresIn) * time.Second).Before(time.Now())
}

// ValidatePKCE : PKCEの検証を行う
func (data *AuthorizeData) ValidatePKCE(verifier string) (bool, error) {
	if len(data.CodeChallenge) == 0 {
		return true, nil
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
		return echo.NewHTTPError(http.StatusBadRequest, err) //普通は起こらないはず
	}

	if len(req.ClientID) == 0 {

	}
}
