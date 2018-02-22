package oauth2

import "github.com/satori/go.uuid"

// AccessScope : クライアントのスコープ
//
// AccessScopeに使用可能な文字のASCIIコードは次の通りです。
//
// %x21, %x23-5B, %x5D-7E
//
// /と"は使えません。
type AccessScope string

// Client : OAuth2.0クライアント構造体
type Client struct {
	ID           string
	Name         string
	Description  string
	Confidential bool
	CreatorID    uuid.UUID
	Secret       string
	RedirectURI  string
	Scope        []AccessScope
}
