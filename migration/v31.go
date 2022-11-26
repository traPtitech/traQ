package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v31 お気に入りスタンプパーミッション削除（削除忘れ）
func v31() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "31",
		Migrate: func(db *gorm.DB) error {
			removedPermissions := []string{
				"get_favorite_stamp",
				"edit_favorite_stamp",
			}
			for _, perm := range removedPermissions {
				if err := db.Delete(&v31RolePermission{}, &v31RolePermission{Permission: perm}).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// v31RolePermission ロール権限構造体
type v31RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

// TableName RolePermission構造体のテーブル名
func (*v31RolePermission) TableName() string {
	return "user_role_permissions"
}
