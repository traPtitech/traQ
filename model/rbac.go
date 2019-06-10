package model

// UserRole ユーザーロール構造体
type UserRole struct {
	Name         string            `gorm:"type:varchar(30);not null;primary_key"`
	OAuth2Scope  bool              `gorm:"type:boolean;not null;default:false"`
	Inheritances []RoleInheritance `gorm:"association_autoupdate:false;association_autocreate:false;foreignkey:Role"`
	Permissions  []RolePermission  `gorm:"association_autoupdate:false;association_autocreate:false;foreignkey:Role"`
	System       bool              `gorm:"type:boolean;not null;default:false"`
}

// TableName UserDefinedRole構造体のテーブル名
func (*UserRole) TableName() string {
	return "user_roles"
}

// RoleInheritance ロール継承関係構造体
type RoleInheritance struct {
	Role    string `gorm:"type:varchar(30);not null;primary_key"`
	SubRole string `gorm:"type:varchar(30);not null;primary_key"`
}

// TableName RoleInheritance構造体のテーブル名
func (*RoleInheritance) TableName() string {
	return "user_role_inheritances"
}

// RolePermission ロール権限構造体
type RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

// TableName RolePermission構造体のテーブル名
func (*RolePermission) TableName() string {
	return "user_role_permissions"
}
