package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"runtime"
	"testing"
)

func TestSourceLocation(t *testing.T) {
	t.Parallel()

	assert.Nil(t, newSourceLocation(0, "", 0, false))

	sl := SourceLocation(runtime.Caller(0)).Interface.(*sourceLocation)

	assert.Contains(t, sl.File, "logging/source_location_test.go")
	assert.Equal(t, "15", sl.Line)
	assert.Equal(t, "github.com/traPtitech/traQ/logging.TestSourceLocation", sl.Function)
}

func TestSource_MarshalLogObject(t *testing.T) {
	t.Parallel()

	sl := &sourceLocation{
		File:     "test1",
		Line:     "test2",
		Function: "test3",
	}

	enc := zapcore.NewMapObjectEncoder()

	if assert.NoError(t, sl.MarshalLogObject(enc)) {
		assert.EqualValues(t, sl.File, enc.Fields["file"])
		assert.EqualValues(t, sl.Line, enc.Fields["line"])
		assert.EqualValues(t, sl.Function, enc.Fields["function"])
	}
}
