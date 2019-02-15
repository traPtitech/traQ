package model

import (
	"github.com/stretchr/testify/assert"
	"strings"
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

func TestClipFolder_Validate(t *testing.T) {
	t.Parallel()

	assert.Error(t, (&ClipFolder{Name: ""}).Validate())
	assert.Error(t, (&ClipFolder{Name: strings.Repeat("a", 31)}).Validate())
	assert.NoError(t, (&ClipFolder{Name: "OK"}).Validate())
}
