package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMute_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "mutes", (&Mute{}).TableName())
}
