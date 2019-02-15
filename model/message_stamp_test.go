package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages_stamps", (&MessageStamp{}).TableName())
}
