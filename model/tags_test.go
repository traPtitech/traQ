package model

import (
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
