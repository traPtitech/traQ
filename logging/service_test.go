package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestServiceContext(t *testing.T) {
	t.Parallel()

	name := "traq"
	ver := "v1.2.3.abcdef"

	sc := ServiceContext(name, ver).Interface.(*serviceContext)

	assert.Equal(t, name, sc.Name)
	assert.Equal(t, ver, sc.Version)
}

func TestServiceContext_MarshalLogObject(t *testing.T) {
	t.Parallel()

	name := "traq"
	ver := "v1.2.3.abcdef"

	sc := &serviceContext{
		Name:    name,
		Version: ver,
	}

	enc := zapcore.NewMapObjectEncoder()

	if assert.NoError(t, sc.MarshalLogObject(enc)) {
		assert.EqualValues(t, name, enc.Fields["service"])
		assert.EqualValues(t, ver, enc.Fields["version"])
	}
}
