package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_roles", (&UserRole{}).TableName())
}

func TestRolePermission_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_role_permissions", (&RolePermission{}).TableName())
}
