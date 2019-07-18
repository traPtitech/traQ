package sessions

const (
	// CookieName セッションクッキー名
	CookieName     = "r_session"
	tableName      = "r_sessions"
	cacheSize      = 4096
	mutexSize      = 1024
	sessionMaxAge  = 60 * 60 * 24 * 14 // 2 weeks
	sessionKeepAge = 60 * 60 * 24 * 14 // 2 weeks
)
