package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPinTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pins", (&Pin{}).TableName())
}
