package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevice_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "devices", (&Device{}).TableName())
}
