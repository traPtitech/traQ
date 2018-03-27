package oauth2

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
	"regexp"
	"time"
)

var pkceStringValidator = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")

// AuthorizeData : Authorization Code Grant用の認可データ構造体
type AuthorizeData struct {
	Code                string
	ClientID            string
	UserID              uuid.UUID
	CreatedAt           time.Time
	ExpiresIn           int
	RedirectURI         string
	Scopes              scope.AccessScopes
	OriginalScopes      scope.AccessScopes
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
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
