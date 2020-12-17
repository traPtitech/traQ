package migration

import (
	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
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
	Role       string `gorm:"type:varchar(30);not null;primary_key"`
	Permission string `gorm:"type:varchar(30);not null;primary_key"`
}

func (*v22RolePermission) TableName() string {
	return "user_role_permissions"
}
