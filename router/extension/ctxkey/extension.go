package ctxkey

// CtxKey context.Context用のキータイプ
type CtxKey int

const (
	// UserID ユーザーUUIDキー
	UserID CtxKey = iota
)
