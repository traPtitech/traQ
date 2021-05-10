package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v22 BOTへのWebRTCパーミッションの付与
func v22() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "22",
		Migrate: func(db *gorm.DB) error {
			addedRolePermissions := map[string][]string{
				"bot": {
					"web_rtc",
				},
			}
			for role, perms := range addedRolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v22RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

type v22RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v22RolePermission) TableName() string {
	return "user_role_permissions"
}
