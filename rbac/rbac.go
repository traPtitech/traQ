package rbac

// RBAC Role-based Access Controllerインターフェース
type RBAC interface {
	// IsGranted 指定したロールで指定した権限が許可されているかどうか
	IsGranted(role string, perm Permission) bool
	// IsAllGranted 指定したロール全てで指定した権限が許可されているかどうか
	IsAllGranted(roles []string, perm Permission) bool
	// IsAnyGranted 指定したロールのいずれかで指定した権限が許可されているかどうか
	IsAnyGranted(roles []string, perm Permission) bool
	// Reload 設定を読み込みます
	Reload() error
	// IsOAuth2Scope 指定したロールがOAuth2Scopeかどうか
	IsOAuth2Scope(v string) bool
	// IsValidRole 有効なロールかどうか
	IsValidRole(v string) bool
	// GetGrantedPermissions 指定したロールに与えられている全ての権限を取得します
	GetGrantedPermissions(role string) Permissions
}

// Permission パーミッション
type Permission string

// Name パーミッション名
func (p Permission) Name() string {
	return string(p)
}

// Permissions パーミッションセット
type Permissions map[Permission]bool

// Add セットに権限を追加します
func (set Permissions) Add(p Permission) {
	set[p] = true
}

// Remove セットから権限を削除します
func (set Permissions) Remove(p Permission) {
	delete(set, p)
}

// Has セットに指定した権限が含まれているかどうか
func (set Permissions) Has(p Permission) bool {
	return set[p]
}

// Array セットの権限の配列を返します
func (set Permissions) Array() []Permission {
	result := make([]Permission, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

// Role ロールインターフェース
type Role interface {
	Name() string
	IsGranted(p Permission) bool
	IsOAuth2Scope() bool
	Permissions() Permissions
}

// Roles ロールセット
type Roles map[string]Role

// Add セットにロールを追加します
func (roles Roles) Add(role Role) {
	roles[role.Name()] = role
}

// IsGranted セットで指定した権限が許可されているかどうか
func (roles Roles) IsGranted(p Permission) bool {
	for _, v := range roles {
		if v.IsGranted(p) {
			return true
		}
	}
	return false
}

// HasAndIsGranted
func (roles Roles) HasAndIsGranted(r string, p Permission) bool {
	set, ok := roles[r]
	if !ok {
		return false
	}
	return set.IsGranted(p)
}
