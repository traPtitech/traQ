package role

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/rbac/permission"
)

// GetSystemRoles システム定義ロールのRolesを返します
func GetSystemRoles() Roles {
	return Roles{
		Admin: &systemRole{
			name:        Admin,
			oauth2Scope: false,
			permissions: permission.PermissionsFromArray(permission.List),
		},
		User: &systemRole{
			name:        User,
			oauth2Scope: false,
			permissions: permission.PermissionsFromArray(userPerms),
		},
		Read: &systemRole{
			name:        Read,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(readPerms),
		},
		Write: &systemRole{
			name:        Write,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(writePerms),
		},
		Bot: &systemRole{
			name:        Bot,
			oauth2Scope: false,
			permissions: permission.PermissionsFromArray(botPerms),
		},
		ManageBot: &systemRole{
			name:        ManageBot,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(manageBotPerms),
		},
		OpenID: &systemRole{
			name:        OpenID,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(openIDPerms),
		},
		Profile: &systemRole{
			name:        Profile,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(profilePerms),
		},
		Client: &systemRole{
			name:        Client,
			oauth2Scope: true,
			permissions: permission.PermissionsFromArray(clientPerms),
		},
	}
}

func SystemRoleModels() []*model.UserRole {
	roles := GetSystemRoles()
	result := make([]*model.UserRole, 0, len(roles))
	for _, role := range roles {
		m := model.UserRole{
			Name:        role.Name(),
			Oauth2Scope: role.(*systemRole).oauth2Scope,
			System:      true,
		}
		if role.Name() != Admin {
			m.Permissions = convertRolePermissions(role.Name(), role.Permissions())
		}
		result = append(result, &m)
	}
	return result
}

func convertRolePermissions(role string, perms permission.Permissions) []model.RolePermission {
	result := make([]model.RolePermission, 0, len(perms))
	for p := range perms {
		result = append(result, model.RolePermission{
			Role:       role,
			Permission: p.Name(),
		})
	}
	return result
}

type systemRole struct {
	name        string
	oauth2Scope bool
	permissions permission.Permissions
}

func (r *systemRole) Name() string {
	return r.name
}

func (r *systemRole) IsGranted(p permission.Permission) bool {
	return r.permissions.Contains(p)
}

func (r *systemRole) Permissions() permission.Permissions {
	return r.permissions
}

// Role ロールインターフェース
type Role interface {
	Name() string
	IsGranted(p permission.Permission) bool
	Permissions() permission.Permissions
}

// Roles ロールセット
type Roles map[string]Role

// Add セットにロールを追加します
func (roles Roles) Add(role Role) {
	roles[role.Name()] = role
}

// IsGranted セットで指定した権限が許可されているかどうか
func (roles Roles) IsGranted(p permission.Permission) bool {
	for _, v := range roles {
		if v.IsGranted(p) {
			return true
		}
	}
	return false
}

// HasAndIsGranted セットが指定したロールを持ち、そのロールに指定した権限が許可されているかどうか
func (roles Roles) HasAndIsGranted(r string, p permission.Permission) bool {
	set, ok := roles[r]
	if !ok {
		return false
	}
	return set.IsGranted(p)
}
