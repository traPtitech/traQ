package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDevice_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "devices", (&Device{}).TableName())
}
