package permission

import (
	"github.com/traPtitech/traQ/rbac"
)

const (
	// GetHeartbeat : ハートビート取得権限
	GetHeartbeat = rbac.Permission("get_heartbeat")
	// PostHeartbeat : ハートビート送信権限
	PostHeartbeat = rbac.Permission("post_heartbeat")
)
