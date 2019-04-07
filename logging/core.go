package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type driverConfig struct {
	ServiceName    string
	ServiceVersion string
}

type core struct {
	zapcore.Core
	config driverConfig
}

// With adds structured context to the Core.
func (c *core) With(fields []zap.Field) zapcore.Core {
	return &core{
		Core:   c.Core.With(fields),
		config: c.config,
	}
}

// Check determines whether the supplied Entry should be logged (using the
// embedded LevelEnabler and possibly some extra logic). If the entry
// should be logged, the Core adds itself to the CheckedEntry and returns
// the result.
//
// Callers must use Check before calling Write.
func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to their destination.
//
// If called, Write should always log the Entry and Fields; it should not
// replicate the logic of Check.
func (c *core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	fields = c.withSourceLocation(ent, fields)
	fields = c.withServiceContext(c.config.ServiceName, c.config.ServiceVersion, fields)
	if zapcore.ErrorLevel.Enabled(ent.Level) {
		fields = c.withErrorReport(ent, fields)
	}
	return c.Core.Write(ent, fields)
}

// Sync flushes buffered logs (if any).
func (c *core) Sync() error {
	return c.Core.Sync()
}

func (c *core) withSourceLocation(ent zapcore.Entry, fields []zapcore.Field) []zapcore.Field {
	for i := range fields {
		if fields[i].Key == sourceLocationKey {
			return fields
		}
	}

	if !ent.Caller.Defined {
		return fields
	}

	return append(fields, SourceLocation(ent.Caller.PC, ent.Caller.File, ent.Caller.Line, true))
}

func (c *core) withServiceContext(name, version string, fields []zapcore.Field) []zapcore.Field {
	for i := range fields {
		if fields[i].Key == serviceContextKey {
			return fields
		}
	}

	return append(fields, ServiceContext(name, version))
}

func (c *core) withErrorReport(ent zapcore.Entry, fields []zapcore.Field) []zapcore.Field {
	for i := range fields {
		if fields[i].Key == contextKey {
			return fields
		}
	}

	if !ent.Caller.Defined {
		return fields
	}

	return append(fields, ErrorReport(ent.Caller.PC, ent.Caller.File, ent.Caller.Line, true))
}
