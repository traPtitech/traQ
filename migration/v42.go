package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v42 get_my_stamp_recommendationsパーミッションの追加とmessages_stampsテーブルへの (user_id, updated_at) の複合インデックスの追加
func v42() *gormigrate.Migration {
	rolePermissions := map[string][]string{
		"user": {
			"get_my_stamp_recommendations",
		},
		"bot": {
			"get_my_stamp_recommendations",
		},
		"read": {
			"get_my_stamp_recommendations",
		},
	}

	return &gormigrate.Migration{
		ID: "42",
		Migrate: func(db *gorm.DB) error {
			for role, perms := range rolePermissions {
				for _, perm := range perms {
					if err := db.Create(&v42RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return db.Exec("CREATE INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps (user_id, updated_at)").Error
		},
		Rollback: func(db *gorm.DB) error {
			for role, perms := range rolePermissions {
				for _, perm := range perms {
					if err := db.Delete(&v42RolePermission{Role: role, Permission: perm}).Error; err != nil {
						return err
					}
				}
			}
			return db.Exec("DROP INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps").Error
		},
	}
}

type v42RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v42RolePermission) TableName() string {
	return "user_role_permissions"
}
