package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"
	"time"
)

func TestEncodeLevel(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		lvl  zapcore.Level
		want string
	}{
		{zapcore.DebugLevel, "DEBUG"},
		{zapcore.InfoLevel, "INFO"},
		{zapcore.WarnLevel, "WARNING"},
		{zapcore.ErrorLevel, "ERROR"},
		{zapcore.DPanicLevel, "CRITICAL"},
		{zapcore.PanicLevel, "ALERT"},
		{zapcore.FatalLevel, "EMERGENCY"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			enc := &sliceArrayEncoder{}
			encodeLevel(tt.lvl, enc)

			if assert.Len(t, enc.elems, 1) {
				assert.Equal(t, enc.elems[0].(string), tt.want)
			}
		})
	}
}

func TestRFC3339NanoTimeEncoder(t *testing.T) {
	t.Parallel()

	ts := time.Date(2020, 12, 3, 4, 56, 78, 910111, time.UTC)

	enc := &sliceArrayEncoder{}
	rfc3339NanoTimeEncoder(ts, enc)

	if assert.Len(t, enc.elems, 1) {
		assert.Equal(t, ts.Format(time.RFC3339Nano), enc.elems[0].(string))
	}
}
