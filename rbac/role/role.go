package role

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

func SystemRoles() []*model.UserDefinedRole {
	return []*model.UserDefinedRole{
		{
			Name:        Admin,
			OAuth2Scope: true,
		},
		{
			Name:        User,
			OAuth2Scope: true,
			Permissions: convertRolePermissions(User, userPerms),
		},
		{
			Name:        Read,
			OAuth2Scope: true,
			Permissions: convertRolePermissions(Read, readPerms),
		},
		{
			Name:        Write,
			OAuth2Scope: true,
			Permissions: convertRolePermissions(Write, writePerms),
		},
		{
			Name:        Bot,
			OAuth2Scope: true,
			Permissions: convertRolePermissions(Bot, botPerms),
		},
		{
			Name:        ManageBot,
			OAuth2Scope: true,
			Permissions: convertRolePermissions(ManageBot, manageBotPerms),
		},
	}
}

func convertRolePermissions(role string, perms []rbac.Permission) []model.RolePermission {
	result := make([]model.RolePermission, len(perms))
	for i, v := range perms {
		result[i] = model.RolePermission{
			Role:       role,
			Permission: v.Name(),
		}
	}
	return result
}
