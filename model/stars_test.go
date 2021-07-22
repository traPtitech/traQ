package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStar_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stars", (&Star{}).TableName())
}
