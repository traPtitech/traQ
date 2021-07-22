package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "messages_stamps", (&MessageStamp{}).TableName())
}
