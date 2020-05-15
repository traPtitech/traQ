package rbac

import "github.com/traPtitech/traQ/service/rbac/permission"

// RBAC Role-based Access Controllerインターフェース
type RBAC interface {
	// IsGranted 指定したロールで指定した権限が許可されているかどうか
	IsGranted(role string, perm permission.Permission) bool
	// IsAllGranted 指定したロール全てで指定した権限が許可されているかどうか
	IsAllGranted(roles []string, perm permission.Permission) bool
	// IsAnyGranted 指定したロールのいずれかで指定した権限が許可されているかどうか
	IsAnyGranted(roles []string, perm permission.Permission) bool
	// GetGrantedPermissions 指定したロールに与えられている全ての権限を取得します
	GetGrantedPermissions(role string) []permission.Permission
}
