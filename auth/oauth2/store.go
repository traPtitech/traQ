package oauth2

import "github.com/satori/go.uuid"

// Store : OAuth2用の各種データのストアインターフェイス
type Store interface {
	ClientStore
	AuthorizeStore
	TokenStore
}

// ClientStore OAuth2用のクライアントストアインターフェイス
type ClientStore interface {
	GetClient(id string) (*Client, error)
	GetClientsByUser(userID uuid.UUID) ([]*Client, error)
	SaveClient(client *Client) error
	UpdateClient(client *Client) error
	DeleteClient(id string) error
}

// AuthorizeStore OAuth2用の認可コードストアインターフェイス
type AuthorizeStore interface {
	SaveAuthorize(data *AuthorizeData) error
	GetAuthorize(code string) (*AuthorizeData, error)
	DeleteAuthorize(code string) error
}

// TokenStore OAuth2用のトークンストアインターフェイス
type TokenStore interface {
	SaveToken(token *Token) error
	GetTokenByID(id uuid.UUID) (*Token, error)
	GetTokenByAccess(access string) (*Token, error)
	DeleteTokenByAccess(access string) error
	GetTokenByRefresh(refresh string) (*Token, error)
	DeleteTokenByRefresh(refresh string) error
	GetTokensByUser(userID uuid.UUID) ([]*Token, error)
	DeleteTokenByUser(userID uuid.UUID) error
	DeleteTokenByClient(clientID string) error
}
