package scope

import (
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/rbac/role"
	"strings"
)

// AccessScope : クライアントのスコープ
//
// AccessScopeに使用可能な文字のASCIIコードは次の通りです。
//
// %x21, %x23-5B, %x5D-7E
//
// /と"は使えません。
type AccessScope string

// AccessScopes : AccessScopeのスライス
type AccessScopes []AccessScope

const (
	// OpenID : OpenID Connect用
	OpenID AccessScope = "openid"
	// Profile : OpenID Connect用
	Profile AccessScope = "profile"

	// Read : 読み込み権限
	Read AccessScope = "read"
	// PrivateRead : プライベートなチャンネルの読み込み権限
	PrivateRead AccessScope = "private_read" //TODO
	// Write : 書き込み権限
	Write AccessScope = "write"
	// PrivateWrite : プライベートなチャンネルの書き込み権限
	PrivateWrite AccessScope = "private_write" //TODO
	// Bot Botユーザー
	Bot AccessScope = "bot"
)

var list = map[AccessScope]gorbac.Role{
	OpenID:       nil,
	Profile:      nil,
	Read:         role.ReadUser,
	PrivateRead:  role.PrivateReadUser,
	Write:        role.WriteUser,
	PrivateWrite: role.PrivateWriteUser,
	Bot:          role.Bot,
}

// Valid : 有効なスコープ文字列かどうかを返します
func Valid(s AccessScope) bool {
	_, ok := list[s]
	return ok
}

// Contains : AccessScopesに指定したスコープが含まれるかどうかを返します
func (arr AccessScopes) Contains(s AccessScope) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// String : AccessScopesをスペース区切りで文字列に出力します
func (arr AccessScopes) String() string {
	var sa []string
	for _, v := range arr {
		sa = append(sa, string(v))
	}
	return strings.Join(sa, " ")
}

// GenerateRole : スコープからroleを生成します
func (arr AccessScopes) GenerateRole() *role.CompositeRole {
	var roles []gorbac.Role
	for _, v := range arr {
		if r, ok := list[v]; ok && r != nil {
			roles = append(roles, r)
		}
	}

	return role.NewCompositeRole(roles...)
}
