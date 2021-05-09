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
	l *zap.Logger
}

func New(logger *zap.Logger) *L {
	return &L{l: logger}
}

func (gl *L) LogMode(level logger.LogLevel) logger.Interface {
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
		return gl
	}
	return New(gl.l.WithOptions(zap.IncreaseLevel(zapLevel)))
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
