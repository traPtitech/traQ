package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSoundboardItem_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "soundboard_items", (&SoundboardItem{}).TableName())
}
