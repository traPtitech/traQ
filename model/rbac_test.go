package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRole_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_roles", (&UserRole{}).TableName())
}

func TestRoleInheritance_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_role_inheritances", (&RoleInheritance{}).TableName())
}

func TestRolePermission_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "user_role_permissions", (&RolePermission{}).TableName())
}
