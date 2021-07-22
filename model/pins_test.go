package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPinTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pins", (&Pin{}).TableName())
}
