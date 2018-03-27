package oauth2

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
	"time"
)

// Token : OAuth2.0 Access Token構造体
type Token struct {
	ID           uuid.UUID
	ClientID     string
	UserID       uuid.UUID
	RedirectURI  string
	AccessToken  string
	RefreshToken string
	CreatedAt    time.Time
	ExpiresIn    int
	Scopes       scope.AccessScopes
}

// GetAvailableScopes : requestで与えられたスコープのうち、利用可能なものを返します
func (t *Token) GetAvailableScopes(request scope.AccessScopes) (result scope.AccessScopes) {
	for _, s := range request {
		if t.Scopes.Contains(s) {
			result = append(result, s)
		}
	}
	return
}

// IsExpired : 有効期限が切れているかどうか
func (t *Token) IsExpired() bool {
	return t.CreatedAt.Add(time.Duration(t.ExpiresIn) * time.Second).Before(time.Now())
}
