package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// v28 ユーザーグループにアイコンを追加
func v28() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "28",
		Migrate: func(db *gorm.DB) error {
			return db.AutoMigrate(&v28UserGroup{})
		},
	}
}

type v28UserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	Icon        uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`

	Admins   []*v28UserGroupAdmin  `gorm:"constraint:user_group_admins_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:GroupID"`
	Members  []*v28UserGroupMember `gorm:"constraint:user_group_members_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:GroupID"`
	IconFile *v28FileMeta          `gorm:"constraint:user_group_icon_files_id_foreign,OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:Icon"`
}

func (*v28UserGroup) TableName() string {
	return "user_groups"
}

type v28UserGroupMember struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

func (*v28UserGroupMember) TableName() string {
	return "user_group_members"
}

type v28UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
}

func (*v28UserGroupAdmin) TableName() string {
	return "user_group_admins"
}

type v28FileMeta struct {
	ID              uuid.UUID      `gorm:"type:char(36);not null;primaryKey"`
	Name            string         `gorm:"type:text;not null"`
	Mime            string         `gorm:"type:text;not null"`
	Size            int64          `gorm:"type:bigint;not null"`
	CreatorID       optional.UUID  `gorm:"type:char(36);index:idx_files_creator_id_created_at,priority:1"`
	Hash            string         `gorm:"type:char(32);not null"`
	Type            model.FileType `gorm:"type:varchar(30);not null"`
	IsAnimatedImage bool           `gorm:"type:boolean;not null;default:false"`
	ChannelID       optional.UUID  `gorm:"type:char(36);index:idx_files_channel_id_created_at,priority:1"`
	CreatedAt       time.Time      `gorm:"precision:6;index:idx_files_channel_id_created_at,priority:2;index:idx_files_creator_id_created_at,priority:2"`
	DeletedAt       gorm.DeletedAt `gorm:"precision:6"`
}

func (f v28FileMeta) TableName() string {
	return "files"
}
