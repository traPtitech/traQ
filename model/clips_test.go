package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClip_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "clips", (&Clip{}).TableName())
}

func TestClipFolder_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "clip_folders", (&ClipFolder{}).TableName())
}
