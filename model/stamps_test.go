package model

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stamps", (&Stamp{}).TableName())
}

func TestStamp_Validate(t *testing.T) {
	t.Parallel()
	assert.Error(t, (&Stamp{}).Validate())
	assert.NoError(t, (&Stamp{Name: "test"}).Validate())
	assert.NoError(t, (&Stamp{Name: "test-ok_stamp12345"}).Validate())
	assert.Error(t, (&Stamp{Name: "アイウエオ"}).Validate())
	assert.Error(t, (&Stamp{Name: "$$test##"}).Validate())
	assert.Error(t, (&Stamp{Name: "$$test##"}).Validate())
	assert.Error(t, (&Stamp{Name: strings.Repeat("a", 33)}).Validate())
}
