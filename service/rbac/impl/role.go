package impl

import "github.com/traPtitech/traQ/service/rbac"

type role struct {
	name         string
	oauth2       bool
	inheritances rbac.Roles
	permissions  rbac.Permissions
}

func (role *role) Name() string {
	return role.name
}

func (role *role) IsGranted(p rbac.Permission) bool {
	return role.permissions.Has(p) || role.inheritances.IsGranted(p)
}

func (role *role) IsOAuth2Scope() bool {
	return role.oauth2
}

func (role *role) Permissions() rbac.Permissions {
	result := rbac.Permissions{}
	for k := range role.permissions {
		result.Add(k)
	}
	for _, v := range role.inheritances {
		for k := range v.Permissions() {
			result.Add(k)
		}
	}
	return result
}
