package testutils

import (
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
)

type rbacImpl struct {
	roles role.Roles
}

func NewTestRBAC() rbac.RBAC {
	return &rbacImpl{
		roles: role.GetSystemRoles(),
	}
}

func (rbac *rbacImpl) Reload() error {
	return nil
}

func (rbac *rbacImpl) IsGranted(r string, p permission.Permission) bool {
	if r == role.Admin {
		return true
	}
	return rbac.roles.HasAndIsGranted(r, p)
}

func (rbac *rbacImpl) IsAllGranted(roles []string, perm permission.Permission) bool {
	for _, role := range roles {
		if !rbac.IsGranted(role, perm) {
			return false
		}
	}
	return true
}

func (rbac *rbacImpl) IsAnyGranted(roles []string, perm permission.Permission) bool {
	for _, role := range roles {
		if rbac.IsGranted(role, perm) {
			return true
		}
	}
	return false
}

func (rbac *rbacImpl) GetGrantedPermissions(roleName string) []permission.Permission {
	ro, ok := rbac.roles[roleName]
	if ok {
		return ro.Permissions().Array()
	}
	return nil
}
