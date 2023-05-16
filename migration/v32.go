package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// ユーザーの表示名上限を32文字に
func v32() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "32",
		Migrate: func(db *gorm.DB) error {
			if err := db.Exec("UPDATE `users` SET `display_name` = LEFT(`display_name`, 32) WHERE CHAR_LENGTH(`display_name`) > 32").Error; err != nil {
				return err
			}
			return db.AutoMigrate(&v32User{})
		},
	}
}

// v32User ユーザー構造体
type v32User struct {
	ID          uuid.UUID            `gorm:"type:char(36);not null;primaryKey"`
	Name        string               `gorm:"type:varchar(32);not null;unique"`
	DisplayName string               `gorm:"type:varchar(32);not null;default:''"`
	Password    string               `gorm:"type:char(128);not null;default:''"`
	Salt        string               `gorm:"type:char(128);not null;default:''"`
	Icon        uuid.UUID            `gorm:"type:char(36);not null"`
	Status      v32UserAccountStatus `gorm:"type:tinyint;not null;default:0"`
	Bot         bool                 `gorm:"type:boolean;not null;default:false"`
	Role        string               `gorm:"type:varchar(30);not null;default:'user'"`
	CreatedAt   time.Time            `gorm:"precision:6"`
	UpdatedAt   time.Time            `gorm:"precision:6"`
}

type (
	v32UserAccountStatus int
)

// TableName User構造体のテーブル名
func (*v32User) TableName() string {
	return "users"
}
