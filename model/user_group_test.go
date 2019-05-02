package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserGroup_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_groups", (&UserGroup{}).TableName())
}

func TestUserGroupMember_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_group_members", (&UserGroupMember{}).TableName())
}
