package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"strconv"
)

const contextKey = "context"

// ErrorReport Stackdriver logging context Field
func ErrorReport(pc uintptr, file string, line int, ok bool) zap.Field {
	return zap.Object(contextKey, newReportContext(pc, file, line, ok))
}

type reportLocation struct {
	File     string `json:"filePath"`
	Line     string `json:"lineNumber"`
	Function string `json:"functionName"`
}

// MarshalLogObject implements zapcore.ObjectMarshaller interface.
func (l reportLocation) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("filePath", l.File)
	enc.AddString("lineNumber", l.Line)
	enc.AddString("functionName", l.Function)
	return nil
}

type reportContext struct {
	ReportLocation reportLocation `json:"reportLocation"`
}

// MarshalLogObject implements zapcore.ObjectMarshaller interface.
func (c reportContext) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return enc.AddObject("reportLocation", c.ReportLocation)
}

func newReportContext(pc uintptr, file string, line int, ok bool) *reportContext {
	if !ok {
		return nil
	}

	context := &reportContext{
		ReportLocation: reportLocation{
			File: file,
			Line: strconv.Itoa(line),
		},
	}

	if fn := runtime.FuncForPC(pc); fn != nil {
		context.ReportLocation.Function = fn.Name()
	}

	return context
}
