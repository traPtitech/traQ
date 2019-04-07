package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const serviceContextKey = "serviceContext"

type serviceContext struct {
	Name    string `json:"service"`
	Version string `json:"version"`
}

// ServiceContext Stackdriver logging serviceContext Field
func ServiceContext(name, version string) zap.Field {
	return zap.Object(serviceContextKey, &serviceContext{
		Name:    name,
		Version: version,
	})
}

// MarshalLogObject implements zapcore.ObjectMarshaller interface.
func (sc serviceContext) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("service", sc.Name)
	enc.AddString("version", sc.Version)
	return nil
}
