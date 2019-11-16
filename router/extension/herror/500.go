package herror

import (
	"fmt"
	"github.com/traPtitech/traQ/logging"
	"go.uber.org/zap"
	"runtime"
	"runtime/debug"
)

type InternalError struct {
	Err    error
	Stack  []byte
	Fields []zap.Field
}

func (i *InternalError) Error() string {
	return fmt.Sprintf("%s\n%s", i.Err.Error(), i.Stack)
}

func InternalServerError(err error) error {
	return &InternalError{
		Err:    err,
		Stack:  debug.Stack(),
		Fields: []zap.Field{logging.ErrorReport(runtime.Caller(1)), zap.Error(err)},
	}
}
