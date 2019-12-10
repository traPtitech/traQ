package herror

import (
	"fmt"
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"runtime"
	"runtime/debug"
)

// InternalError 内部エラー
type InternalError struct {
	// Err エラー
	Err error
	// Stack スタックトレース
	Stack []byte
	// Fields zapログ用フィールド
	Fields []zap.Field
}

func (i *InternalError) Error() string {
	return fmt.Sprintf("%s\n%s", i.Err.Error(), i.Stack)
}

func InternalServerError(err error) error {
	return &InternalError{
		Err:    err,
		Stack:  debug.Stack(),
		Fields: []zap.Field{zapdriver.ErrorReport(runtime.Caller(1)), zap.Error(err)},
	}
}
