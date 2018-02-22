package oauth2

import "github.com/satori/go.uuid"

// ClientScope : クライアントのスコープ
//
// ClientScopeに使用可能な文字のASCIIコードは次の通りです。
//
// %x21, %x23-5B, %x5D-7E
//
// /と"は使えません。
type ClientScope string

// Client : OAuth2.0クライアント構造体
type Client struct {
	ID          string
	Name        string
	Description string
	CreatorID   uuid.UUID
	Secret      string
	RedirectURI string
	Scope       []ClientScope
}
