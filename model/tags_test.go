package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tags", (&Tag{}).TableName())
}

func TestUsersTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_tags", (&UsersTag{}).TableName())
}

func TestTag_Validate(t *testing.T) {
	t.Parallel()

	assert.Error(t, (&Tag{Name: ""}).Validate())
	assert.Error(t, (&Tag{Name: strings.Repeat("a", 31)}).Validate())
	assert.NoError(t, (&Tag{Name: "aa"}).Validate())
}
