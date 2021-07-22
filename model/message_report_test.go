package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageReport_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "message_reports", (&MessageReport{}).TableName())
}
