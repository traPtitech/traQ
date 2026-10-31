package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func v20261023() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20261023",
		Migrate: func(db *gorm.DB) error {
			return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&v20261023RolePermission{
				Role: "openid", Permission: "get_oidc_userinfo",
			}).Error
		},
	}
}

type v20261023RolePermission struct {
	Role       string `gorm:"type:varchar(30);not null;primaryKey"`
	Permission string `gorm:"type:varchar(30);not null;primaryKey"`
}

func (*v20261023RolePermission) TableName() string {
	return "user_role_permissions"
}
