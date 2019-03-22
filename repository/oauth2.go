package repository

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// OAuth2Repository OAuth2用リポジトリ
type OAuth2Repository interface {
	ClientStore
	AuthorizeStore
	TokenStore
}

// ClientStore OAuth2用のクライアントストアインターフェイス
type ClientStore interface {
	GetClient(id string) (*model.OAuth2Client, error)
	GetClientsByUser(userID uuid.UUID) ([]*model.OAuth2Client, error)
	SaveClient(client *model.OAuth2Client) error
	UpdateClient(client *model.OAuth2Client) error
	DeleteClient(id string) error
}

// AuthorizeStore OAuth2用の認可コードストアインターフェイス
type AuthorizeStore interface {
	SaveAuthorize(data *model.OAuth2Authorize) error
	GetAuthorize(code string) (*model.OAuth2Authorize, error)
	DeleteAuthorize(code string) error
}

// TokenStore OAuth2用のトークンストアインターフェイス
type TokenStore interface {
	IssueToken(client *model.OAuth2Client, userID uuid.UUID, redirectURI string, scope model.AccessScopes, expire int, refresh bool) (*model.OAuth2Token, error)
	GetTokenByID(id uuid.UUID) (*model.OAuth2Token, error)
	DeleteTokenByID(id uuid.UUID) error
	GetTokenByAccess(access string) (*model.OAuth2Token, error)
	DeleteTokenByAccess(access string) error
	GetTokenByRefresh(refresh string) (*model.OAuth2Token, error)
	DeleteTokenByRefresh(refresh string) error
	GetTokensByUser(userID uuid.UUID) ([]*model.OAuth2Token, error)
	DeleteTokenByUser(userID uuid.UUID) error
	DeleteTokenByClient(clientID string) error
}
