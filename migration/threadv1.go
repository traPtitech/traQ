package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// threadV1 スレッド通知を管理するテーブルを追加
func threadv1() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "thread1",
		Migrate: func(db *gorm.DB) error {

			if err := db.AutoMigrate(&threadv1UserSubscribeThread{}); err != nil {
				return err
			}

			if err := db.AutoMigrate(&threadv1Channel{}); err != nil {
				return err
			}

			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"users_subscribe_threads", "users_subscribe_threads_user_id_users_id_foreign", "user_id", "users(id)", "CASCADE", "CASCADE"},
				{"users_subscribe_threads", "users_subscribe_threads_channel_id_channels_id_foreign", "channel_id", "channels(id)", "CASCADE", "CASCADE"},
			}

			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			return nil

		},
	}
}

type threadv1UserSubscribeThread struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Mark      bool      `gorm:"type:boolean;not null;default:false"`
	Notify    bool      `gorm:"type:boolean;not null;default:false"`
}

type threadv1Channel struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey;index:idx_channel_channels_id_is_public_is_forced,priority:1"`
	Name      string         `gorm:"type:varchar(20);not null;uniqueIndex:name_parent"`
	ParentID  uuid.UUID      `gorm:"type:char(36);not null;uniqueIndex:name_parent"`
	Topic     string         `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"`
	IsForced  bool           `gorm:"type:boolean;not null;default:false;index:idx_channel_channels_id_is_public_is_forced,priority:3"`
	IsPublic  bool           `gorm:"type:boolean;not null;default:false;index:idx_channel_channels_id_is_public_is_forced,priority:2"`
	IsVisible bool           `gorm:"type:boolean;not null;default:false"`
	IsThread  bool           `gorm:"type:boolean;not null;default:false"`
	CreatorID uuid.UUID      `gorm:"type:char(36);not null"`
	UpdaterID uuid.UUID      `gorm:"type:char(36);not null"`
	CreatedAt time.Time      `gorm:"precision:6"`
	UpdatedAt time.Time      `gorm:"precision:6"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"`

	ChildrenID []uuid.UUID `gorm:"-"`
}

func (*threadv1UserSubscribeThread) TableName() string {
	return "users_subscribe_threads"
}
