package role

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
)

func SystemRoles() []*model.UserRole {
	return []*model.UserRole{
		{
			Name:        Admin,
			Oauth2Scope: true,
			System:      true,
		},
		{
			Name:        User,
			Oauth2Scope: true,
			Permissions: convertRolePermissions(User, userPerms),
			System:      true,
		},
		{
			Name:        Read,
			Oauth2Scope: true,
			Permissions: convertRolePermissions(Read, readPerms),
			System:      true,
		},
		{
			Name:        Write,
			Oauth2Scope: true,
			Permissions: convertRolePermissions(Write, writePerms),
			System:      true,
		},
		{
			Name:        Bot,
			Oauth2Scope: true,
			Permissions: convertRolePermissions(Bot, botPerms),
			System:      true,
		},
		{
			Name:        ManageBot,
			Oauth2Scope: true,
			Permissions: convertRolePermissions(ManageBot, manageBotPerms),
			System:      true,
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
