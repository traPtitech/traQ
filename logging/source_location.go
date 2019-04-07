package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"strconv"
)

const sourceLocationKey = "logging.googleapis.com/sourceLocation"

// SourceLocation Stackdriver logging sourceLocation Field
func SourceLocation(pc uintptr, file string, line int, ok bool) zap.Field {
	return zap.Object(sourceLocationKey, newSourceLocation(pc, file, line, ok))
}

type sourceLocation struct {
	File     string `json:"file"`
	Line     string `json:"line"`
	Function string `json:"function"`
}

// MarshalLogObject implements zapcore.ObjectMarshaller interface.
func (s sourceLocation) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("file", s.File)
	enc.AddString("line", s.Line)
	enc.AddString("function", s.Function)
	return nil
}

func newSourceLocation(pc uintptr, file string, line int, ok bool) *sourceLocation {
	if !ok {
		return nil
	}

	source := &sourceLocation{
		File: file,
		Line: strconv.Itoa(line),
	}

	if fn := runtime.FuncForPC(pc); fn != nil {
		source.Function = fn.Name()
	}

	return source
}
