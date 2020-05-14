package permission

import (
	"github.com/traPtitech/traQ/service/rbac"
)

const (
	// GetMyTokens 自トークン情報取得権限
	GetMyTokens = rbac.Permission("get_my_tokens")
	// RevokeMyToken 自トークン削除権限
	RevokeMyToken = rbac.Permission("revoke_my_token")
	// GetClients クライアント情報取得権限
	GetClients = rbac.Permission("get_clients")
	// CreateClient 新規クライアント登録権限
	CreateClient = rbac.Permission("create_client")
	// EditMyClient クライアント情報編集権限
	EditMyClient = rbac.Permission("edit_my_client")
	// DeleteMyClient クライアント削除権限
	DeleteMyClient = rbac.Permission("delete_my_client")
	// ManageOthersClient 他人のClientの管理権限
	ManageOthersClient = rbac.Permission("manage_others_client")
)
