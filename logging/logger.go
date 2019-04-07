package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CreateNewLogger ロガーを生成します
func CreateNewLogger(serviceName, serviceVersion string) (*zap.Logger, error) {
	return zapConfig.Build(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return &core{
			Core:   c,
			config: driverConfig{ServiceName: serviceName, ServiceVersion: serviceVersion},
		}
	}))
}
