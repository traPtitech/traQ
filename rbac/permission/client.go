package permission

import "github.com/mikespook/gorbac"

var (
	// GetMyTokens : 自トークン情報取得権限
	GetMyTokens = gorbac.NewStdPermission("get_my_tokens")
	// RevokeMyToken : 自トークン削除権限
	RevokeMyToken = gorbac.NewStdPermission("revoke_my_token")
	// GetClients : クライアント情報取得権限
	GetClients = gorbac.NewStdPermission("get_clients")
	// CreateClient : 新規クライアント登録権限
	CreateClient = gorbac.NewStdPermission("create_client")
	// EditMyClient : クライアント情報編集権限
	EditMyClient = gorbac.NewStdPermission("edit_my_client")
	// DeleteMyClient : クライアント削除権限
	DeleteMyClient = gorbac.NewStdPermission("delete_my_client")
)
