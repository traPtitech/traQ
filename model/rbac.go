package model

// UserRole ユーザーロール構造体
type UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primaryKey"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`

	Inheritances []*UserRole      `gorm:"many2many:user_role_inheritances;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:Name;references:Name;joinForeignKey:Role;joinReferences:SubRole"`
	Permissions  []RolePermission `gorm:"constraint:user_role_permissions_role_user_roles_name_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:Role;references:Name"`
}

// TableName UserRole構造体のテーブル名
func (*UserRole) TableName() string {
	return "user_roles"
}

// RolePermission ロール権限構造体
type RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

// TableName RolePermission構造体のテーブル名
func (*RolePermission) TableName() string {
	return "user_role_permissions"
}
