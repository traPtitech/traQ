package migration

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/utils/optional"
	"gopkg.in/gormigrate.v1"
	"time"
)

// v9 ユーザーテーブル拡張
func v9() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "9",
		Migrate: func(db *gorm.DB) error {
			// UserProfileテーブル生成
			if err := db.AutoMigrate(&v9UserProfile{}).Error; err != nil {
				return err
			}

			// データ移行
			var users []v9OldUser
			if err := db.Find(&users).Error; err != nil {
				return err
			}
			for _, oldUser := range users {
				profile := &v9UserProfile{
					UserID:     oldUser.ID,
					Bio:        "",
					TwitterID:  oldUser.TwitterID,
					LastOnline: oldUser.LastOnline,
					UpdatedAt:  oldUser.UpdatedAt,
				}
				if err := db.Create(profile).Error; err != nil {
					return err
				}
			}

			// 旧カラム削除
			if err := db.Table(v9OldUser{}.TableName()).DropColumn("twitter_id").Error; err != nil {
				return err
			}
			if err := db.Table(v9OldUser{}.TableName()).DropColumn("last_online").Error; err != nil {
				return err
			}

			// 外部キー制約
			foreignKeys := [][5]string{
				{"user_profiles", "user_id", "users(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type v9OldUser struct {
	ID          uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Name        string        `gorm:"type:varchar(32);not null;unique"`
	DisplayName string        `gorm:"type:varchar(64);not null;default:''"`
	Password    string        `gorm:"type:char(128);not null;default:''"`
	Salt        string        `gorm:"type:char(128);not null;default:''"`
	Icon        uuid.UUID     `gorm:"type:char(36);not null"`
	Status      int           `gorm:"type:tinyint;not null;default:0"`
	Bot         bool          `gorm:"type:boolean;not null;default:false"`
	Role        string        `gorm:"type:varchar(30);not null;default:'user'"`
	TwitterID   string        `gorm:"type:varchar(15);not null;default:''"`
	LastOnline  optional.Time `gorm:"precision:6"`
	CreatedAt   time.Time     `gorm:"precision:6"`
	UpdatedAt   time.Time     `gorm:"precision:6"`
}

func (v9OldUser) TableName() string {
	return "users"
}

type v9UserProfile struct {
	UserID     uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Bio        string        `sql:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	TwitterID  string        `gorm:"type:varchar(15);not null;default:''"`
	LastOnline optional.Time `gorm:"precision:6"`
	UpdatedAt  time.Time     `gorm:"precision:6"`
}

func (v9UserProfile) TableName() string {
	return "user_profiles"
}
