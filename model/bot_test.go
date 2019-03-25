package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBot_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bots", (&Bot{}).TableName())
}
