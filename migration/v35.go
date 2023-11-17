package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v35  OIDC実装のため、openid, profile, emailロール、get_oidc_userinfo権限を追加
func v35() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "35",
		Migrate: func(db *gorm.DB) error {
			roles := []v35UserRole{
				{
					Name:        "openid",
					Oauth2Scope: true,
					System:      true,
					Permissions: []v35RolePermission{},
				},
				{
					Name:        "profile",
					Oauth2Scope: true,
					System:      true,
					Permissions: []v35RolePermission{
						{
							Role:       "profile",
							Permission: "get_oidc_userinfo",
						},
					},
				},
				{
					Name:        "email",
					Oauth2Scope: true,
					System:      true,
					Permissions: []v35RolePermission{
						{
							Role:       "profile",
							Permission: "get_oidc_userinfo",
						},
					},
				},
			}
			for _, role := range roles {
				err := db.Create(&role).Error
				if err != nil {
					return err
				}
			}

			rolePermissions := []v35RolePermission{
				{
					Role:       "read",
					Permission: "get_oidc_userinfo",
				},
				{
					Role:       "user",
					Permission: "get_oidc_userinfo",
				},
				{
					Role:       "bot",
					Permission: "get_oidc_userinfo",
				},
				{
					Role:       "manage_bot",
					Permission: "get_oidc_userinfo",
				},
			}
			for _, rp := range rolePermissions {
				err := db.Create(&rp).Error
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type v35UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primaryKey"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`

	Permissions []v35RolePermission `gorm:"constraint:user_role_permissions_role_user_roles_name_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:Role;references:Name"`
}

func (*v35UserRole) TableName() string {
	return "user_roles"
}

type v35RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v35RolePermission) TableName() string {
	return "user_role_permissions"
}
