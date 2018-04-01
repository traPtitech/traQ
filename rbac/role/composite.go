package role

import (
	"github.com/mikespook/gorbac"
	"strings"
)

// CompositeRole : 複合ロール
// gorbacの継承が使いづらい
type CompositeRole struct {
	roles gorbac.Roles
}

// ID returns the role's identity name.
func (r *CompositeRole) ID() string {
	var arr []string
	for k := range r.roles {
		arr = append(arr, k)
	}
	return strings.Join(arr, ",")
}

// Permit returns true if the role has specific permission.
func (r *CompositeRole) Permit(p gorbac.Permission) bool {
	for _, v := range r.roles {
		if v.Permit(p) {
			return true
		}
	}
	return false
}

// NewCompositeRole : 複合ロールを生成します
func NewCompositeRole(roles ...gorbac.Role) *CompositeRole {
	role := &CompositeRole{
		roles: make(gorbac.Roles),
	}
	for _, v := range roles {
		role.roles[v.ID()] = v
	}
	return role
}
