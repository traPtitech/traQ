package permission

import "github.com/mikespook/gorbac"

// GetMetrics メトリクスアクセス権限
var GetMetrics = gorbac.NewStdPermission("get_metrics")
