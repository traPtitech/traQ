package oauth2

import "github.com/satori/go.uuid"

// UserInfo ユーザー情報インターフェイス
type UserInfo interface {
	GetUID() uuid.UUID
	GetName() string
}
