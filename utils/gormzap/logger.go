package gormzap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type L struct {
	l                    *zap.Logger
	parameterizedQueries bool // same as https://gorm.io/ja_JP/docs/logger.html
}

func New(zl *zap.Logger, options ...Option) *L {
	l := &L{l: zl}

	for _, o := range options {
		o(l)
	}

	return l
}

var _ logger.Interface = (*L)(nil)
var _ gorm.ParamsFilter = (*L)(nil)

type Option func(l *L)

func WithParameterizedQueries(enabled bool) Option {
	return func(l *L) {
		l.parameterizedQueries = enabled
	}
}

func (gl L) LogMode(level logger.LogLevel) logger.Interface {
	var zapLevel zapcore.LevelEnabler
	switch level {
	case logger.Silent:
		zapLevel = zap.DPanicLevel
	case logger.Error:
		zapLevel = zap.ErrorLevel
	case logger.Warn:
		zapLevel = zap.WarnLevel
	case logger.Info:
		zapLevel = zap.InfoLevel
	default:
		return &gl
	}

	gl.l = gl.l.WithOptions(zap.IncreaseLevel(zapLevel))

	return &gl
}

func (gl *L) Info(_ context.Context, s string, i ...interface{}) {
	gl.l.Info(fmt.Sprintf(s, i...))
}

func (gl *L) Warn(_ context.Context, s string, i ...interface{}) {
	gl.l.Warn(fmt.Sprintf(s, i...))
}

func (gl *L) Error(_ context.Context, s string, i ...interface{}) {
	gl.l.Error(fmt.Sprintf(s, i...))
}

func (gl *L) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		if rows == -1 {
			gl.l.Error(sql, zap.String("file", utils.FileWithLineNum()), zap.Error(err), zap.Float64("latency(ms)", float64(elapsed.Nanoseconds())/1e6))
		} else {
			gl.l.Error(sql, zap.String("file", utils.FileWithLineNum()), zap.Error(err), zap.Float64("latency(ms)", float64(elapsed.Nanoseconds())/1e6), zap.Int64("rows", rows))
		}
	default:
		sql, rows := fc()
		if rows == -1 {
			gl.l.Debug(sql, zap.String("file", utils.FileWithLineNum()), zap.Float64("latency(ms)", float64(elapsed.Nanoseconds())/1e6))
		} else {
			gl.l.Debug(sql, zap.String("file", utils.FileWithLineNum()), zap.Float64("latency(ms)", float64(elapsed.Nanoseconds())/1e6), zap.Int64("rows", rows))
		}
	}
}

// ParamsFilter implements [(gorm.io/gorm).ParamsFilter]
// https://github.com/go-gorm/gorm/blob/4e34a6d21b63e9a9b701a70be9759e5539bf26e9/logger/logger.go#L192-L198
func (gl *L) ParamsFilter(_ context.Context, sql string, params ...any) (string, []any) {
	if gl.parameterizedQueries {
		return sql, nil
	}
	return sql, params
}
