package scope

import (
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
	// Email : OpenID Connect用
	Email AccessScope = "email"

	// Read : 読み込み権限
	Read AccessScope = "read"
	// PrivateRead : プライベートなチャンネルの読み込み権限
	PrivateRead AccessScope = "private_read"
	// Write : 書き込み権限
	Write AccessScope = "write"
	// PrivateWrite : プライベートなチャンネルの書き込み権限
	PrivateWrite AccessScope = "private_write"
)

var list = map[AccessScope]bool{
	OpenID:       true,
	Profile:      true,
	Email:        true,
	Read:         true,
	PrivateRead:  true,
	Write:        true,
	PrivateWrite: true,
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
