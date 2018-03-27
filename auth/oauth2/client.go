package oauth2

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/auth/oauth2/scope"
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
	Scopes       scope.AccessScopes
}

// GetAvailableScopes : requestで与えられたスコープのうち、利用可能なものを返します
func (c *Client) GetAvailableScopes(request scope.AccessScopes) (result scope.AccessScopes) {
	for _, s := range request {
		if c.Scopes.Contains(s) {
			result = append(result, s)
		}
	}
	return
}
