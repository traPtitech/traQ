package permission

import "github.com/mikespook/gorbac"

var (
	// GetHeartbeat : ハートビート取得権限
	GetHeartbeat = gorbac.NewStdPermission("get_heartbeat")
	// PostHeartbeat : ハートビート送信権限
	PostHeartbeat = gorbac.NewStdPermission("post_heartbeat")
)
