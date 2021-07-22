package herror

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
)

// InternalError 内部エラー
type InternalError struct {
	// Err エラー
	Err error
	// Stack スタックトレース
	Stack []byte
	// Fields zapログ用フィールド
	Fields []zap.Field
	// Panic panicが発生したかどうか
	Panic bool
}

func (i *InternalError) Error() string {
	if i.Panic {
		return fmt.Sprintf("[Panic] %s\n%s", i.Err.Error(), i.Stack)
	}
	return fmt.Sprintf("%s\n%s", i.Err.Error(), i.Stack)
}

func InternalServerError(err error) error {
	return &InternalError{
		Err:    err,
		Stack:  debug.Stack(),
		Fields: []zap.Field{zapdriver.ErrorReport(runtime.Caller(1)), zap.Error(err)},
		Panic:  false,
	}
}

func Panic(err error) error {
	return &InternalError{
		Err:    err,
		Stack:  debug.Stack(),
		Fields: []zap.Field{zapdriver.ErrorReport(runtime.Caller(1)), zap.Error(err)},
		Panic:  true,
	}
}
