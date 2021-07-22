package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgumentError_Error(t *testing.T) {
	t.Parallel()
	msg := "test2"
	assert.Equal(t, msg, ArgError("", msg).Error())
}

func TestArgError(t *testing.T) {
	t.Parallel()

	f := "test1"
	m := "test2"
	err := ArgError(f, m)
	assert.Equal(t, f, err.FieldName)
	assert.Equal(t, m, err.Message)
}

func TestIsArgError(t *testing.T) {
	t.Parallel()
	assert.True(t, IsArgError(ArgError("", "")))
	assert.False(t, IsArgError(nil))
	assert.False(t, IsArgError(ErrAlreadyExists))
}
