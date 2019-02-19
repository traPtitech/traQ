package model

import (
	"github.com/stretchr/testify/assert"
	"strings"
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

func TestUserGroup_Validate(t *testing.T) {
	t.Parallel()

	assert.Error(t, (&UserGroup{Name: ""}).Validate())
	assert.Error(t, (&UserGroup{Name: strings.Repeat("あ", 31)}).Validate())
	assert.Error(t, (&UserGroup{Name: strings.Repeat("a", 31)}).Validate())
	assert.NoError(t, (&UserGroup{Name: "test_group"}).Validate())
}
