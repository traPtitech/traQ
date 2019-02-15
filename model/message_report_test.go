package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageReport_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "message_reports", (&MessageReport{}).TableName())
}
