package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v20261014 qallロールを追加
func v20261014() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20261014",
		Migrate: func(db *gorm.DB) error {
			roles := []v20261014UserRole{
				{
					Name:        "qall",
					Oauth2Scope: true,
					System:      true,
					Permissions: []v20261014RolePermission{
						{
							Role:       "qall",
							Permission: "web_rtc",
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

type v20261014UserRole struct {
	Name        string `gorm:"type:varchar(30);not null;primaryKey"`
	Oauth2Scope bool   `gorm:"type:boolean;not null;default:false"`
	System      bool   `gorm:"type:boolean;not null;default:false"`

	Permissions []v20261014RolePermission `gorm:"constraint:user_role_permissions_role_user_roles_name_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:Role;references:Name"`
}

func (*v20261014UserRole) TableName() string {
	return "user_roles"
}

type v20261014RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v20261014RolePermission) TableName() string {
	return "user_role_permissions"
}
