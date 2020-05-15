package permission

const (
	// GetMyTokens 自トークン情報取得権限
	GetMyTokens = Permission("get_my_tokens")
	// RevokeMyToken 自トークン削除権限
	RevokeMyToken = Permission("revoke_my_token")
	// GetClients クライアント情報取得権限
	GetClients = Permission("get_clients")
	// CreateClient 新規クライアント登録権限
	CreateClient = Permission("create_client")
	// EditMyClient クライアント情報編集権限
	EditMyClient = Permission("edit_my_client")
	// DeleteMyClient クライアント削除権限
	DeleteMyClient = Permission("delete_my_client")
	// ManageOthersClient 他人のClientの管理権限
	ManageOthersClient = Permission("manage_others_client")
)
