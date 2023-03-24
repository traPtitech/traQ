package ctxkey

// ctxKey context.Context用のキータイプ
type ctxKey int

const (
	// UserID ユーザーUUIDキー
	UserID ctxKey = iota
)
