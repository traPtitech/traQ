package oauth2

import (
	"github.com/satori/go.uuid"
	"time"
)

// Token : OAuth2.0 Access Token構造体
type Token struct {
	ClientID     string
	UserID       uuid.UUID
	RedirectURI  string
	AccessToken  string
	RefreshToken string
	CreatedAt    time.Time
	ExpiresIn    int
	Scope        []AccessScope
}
