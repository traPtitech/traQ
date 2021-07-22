package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stamps", (&Stamp{}).TableName())
}
