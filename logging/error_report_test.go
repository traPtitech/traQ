package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"runtime"
	"testing"
)

func TestErrorReport(t *testing.T) {
	t.Parallel()

	assert.Nil(t, newSourceLocation(0, "", 0, false))

	c := ErrorReport(runtime.Caller(0)).Interface.(*reportContext)

	assert.Contains(t, c.ReportLocation.File, "logging/error_report_test.go")
	assert.Equal(t, "15", c.ReportLocation.Line)
	assert.Equal(t, "github.com/traPtitech/traQ/logging.TestErrorReport", c.ReportLocation.Function)
}

func TestReportContext_MarshalLogObject(t *testing.T) {
	t.Parallel()

	c := &reportContext{
		ReportLocation: reportLocation{
			File:     "test1",
			Line:     "test2",
			Function: "test3",
		},
	}

	enc := zapcore.NewMapObjectEncoder()

	if assert.NoError(t, c.MarshalLogObject(enc)) {
		assert.EqualValues(t, c.ReportLocation.File, enc.Fields["reportLocation"].(map[string]interface{})["filePath"])
		assert.EqualValues(t, c.ReportLocation.Line, enc.Fields["reportLocation"].(map[string]interface{})["lineNumber"])
		assert.EqualValues(t, c.ReportLocation.Function, enc.Fields["reportLocation"].(map[string]interface{})["functionName"])
	}
}
