package model

// UserDefinedRole ユーザー定義ロール構造体
type UserDefinedRole struct {
	Name         string            `gorm:"type:varchar(30);not null;primary_key"`
	OAuth2Scope  bool              `gorm:"type:boolean;not null;default:false"`
	Inheritances []RoleInheritance `gorm:"association_autoupdate:false;association_autocreate:false;foreignkey:Role"`
	Permissions  []RolePermission  `gorm:"association_autoupdate:false;association_autocreate:false;foreignkey:Role"`
}

// TableName UserDefinedRole構造体のテーブル名
func (*UserDefinedRole) TableName() string {
	return "user_defined_roles"
}

// RoleInheritance ロール継承関係構造体
type RoleInheritance struct {
	Role    string `gorm:"type:varchar(30);not null;primary_key"`
	SubRole string `gorm:"type:varchar(30);not null;primary_key"`
}

// TableName RoleInheritance構造体のテーブル名
func (*RoleInheritance) TableName() string {
	return "user_defined_role_inheritances"
}

// RolePermission ロール権限構造体
type RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

// TableName RolePermission構造体のテーブル名
func (*RolePermission) TableName() string {
	return "user_defined_role_permissions"
}
