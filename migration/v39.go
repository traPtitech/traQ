package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v39 OAuth Client Credentials Grantの対応のため、clientロールを追加
func v39() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "39",
		Migrate: func(db *gorm.DB) error {
			roles := []v39UserRole{
				{
					Name:        "client",
					Oauth2Scope: false,
					System:      true,
					Permissions: []v39RolePermission{
						{
							Role:       "client",
							Permission: "get_user",
						},
						{
							Role:       "client",
							Permission: "get_user_tag",
						},
						{
							Role:       "client",
							Permission: "get_user_group",
						},
						{
							Role:       "client",
							Permission: "get_stamp",
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
			return nil
		},
	}
}

type v39UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primaryKey"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`

	Permissions []v39RolePermission `gorm:"constraint:user_role_permissions_role_user_roles_name_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:Role;references:Name"`
}

func (*v39UserRole) TableName() string {
	return "user_roles"
}

type v39RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v39RolePermission) TableName() string {
	return "user_role_permissions"
}
