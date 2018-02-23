package oauth2

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/scope"
)

// Client : OAuth2.0クライアント構造体
type Client struct {
	ID           string
	Name         string
	Description  string
	Confidential bool
	CreatorID    uuid.UUID
	Secret       string
	RedirectURI  string
	Scope        scope.AccessScopes
}
