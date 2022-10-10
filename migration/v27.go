package migration

import (
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// v27 Gorm v2移行: FKの追加、FKのリネーム、一部フィールドのデータ型変更、idx_messages_channel_idの削除
func v27() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "27",
		Migrate: func(db *gorm.DB) error {
			deleteIndexes := [][2]string{
				// table name, index name
				{"messages", "idx_messages_channel_id"},
			}
			for _, c := range deleteIndexes {
				if err := db.Migrator().DropIndex(c[0], c[1]); err != nil {
					return err
				}
			}
			deleteForeignKeys := [][2]string{
				// table name, constraint name
				{"user_role_inheritances", "user_role_inheritances_role_user_roles_name_foreign"},
				{"user_role_inheritances", "user_role_inheritances_sub_role_user_roles_name_foreign"},
				{"dm_channel_mappings", "dm_channel_mappings_user1_users_id_foreign"},
				{"dm_channel_mappings", "dm_channel_mappings_user2_users_id_foreign"},
			}
			for _, c := range deleteForeignKeys {
				if err := db.Migrator().DropConstraint(c[0], c[1]); err != nil {
					return err
				}
			}
			foreignKeys := [][6]string{
				// table name, constraint name, field name, references, on delete, on update
				{"user_role_inheritances", "fk_user_role_inheritances_user_role", "role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"user_role_inheritances", "fk_user_role_inheritances_inheritances", "sub_role", "user_roles(name)", "CASCADE", "CASCADE"},
				{"dm_channel_mappings", "dm_channel_mappings_user_one_users_id_foreign", "user1", "users(id)", "CASCADE", "CASCADE"},
				{"dm_channel_mappings", "dm_channel_mappings_user_two_users_id_foreign", "user2", "users(id)", "CASCADE", "CASCADE"},
			}
			for _, c := range foreignKeys {
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s", c[0], c[1], c[2], c[3], c[4], c[5])).Error; err != nil {
					return err
				}
			}

			alterColumns := []struct {
				dst   interface{}
				field string
			}{
				// table, column name
				{&v27BotEventLog{}, "code"},
				{&v27FileMeta{}, "type"},
				{&v27FileThumbnail{}, "width"},
				{&v27FileThumbnail{}, "height"},
				{&v27MessageStamp{}, "count"},
				{&v27OAuth2Authorize{}, "expires_in"},
				{&v27OAuth2Token{}, "expires_in"},
				{&v27OgpCache{}, "id"},
			}
			for _, c := range alterColumns {
				if err := db.Migrator().AlterColumn(c.dst, c.field); err != nil {
					return err
				}
			}

			return db.AutoMigrate(
				&v27UserGroup{},
				&v27UserGroupAdmin{},
				&v27UserGroupMember{},
			)
		},
		Rollback: nil,
	}
}

type v27UserGroup struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`

	// Foreign key 追加
	Admins []*v27UserGroupAdmin `gorm:"constraint:user_group_admins_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:GroupID"`
	// Foreign key 追加
	Members []*v27UserGroupMember `gorm:"constraint:user_group_members_group_id_user_groups_id_foreign,OnUpdate:CASCADE,OnDelete:CASCADE;foreignkey:GroupID"`
}

func (*v27UserGroup) TableName() string {
	return "user_groups"
}

type v27UserGroupMember struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Role    string    `gorm:"type:varchar(100);not null;default:''"`
}

func (*v27UserGroupMember) TableName() string {
	return "user_group_members"
}

type v27UserGroupAdmin struct {
	GroupID uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	UserID  uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
}

func (*v27UserGroupAdmin) TableName() string {
	return "user_group_admins"
}

type v27BotEventLog struct {
	RequestID uuid.UUID          `gorm:"type:char(36);not null;primaryKey"                json:"requestId"`
	BotID     uuid.UUID          `gorm:"type:char(36);not null;index:bot_id_date_time_idx" json:"botId"`
	Event     model.BotEventType `gorm:"type:varchar(30);not null"                         json:"event"`
	Body      string             `gorm:"type:text"                                         json:"-"`
	Error     string             `gorm:"type:text"                                         json:"-"`
	// int(11) -> bigint(20)
	Code     int       `gorm:"not null;default:0"                                json:"code"`
	Latency  int64     `gorm:"not null;default:0"                                json:"-"`
	DateTime time.Time `gorm:"precision:6;index:bot_id_date_time_idx"            json:"dateTime"`
}

func (*v27BotEventLog) TableName() string {
	return "bot_event_logs"
}

type v27FileMeta struct {
	ID        uuid.UUID              `gorm:"type:char(36);not null;primaryKey"`
	Name      string                 `gorm:"type:text;not null"`
	Mime      string                 `gorm:"type:text;not null"`
	Size      int64                  `gorm:"type:bigint;not null"`
	CreatorID optional.Of[uuid.UUID] `gorm:"type:char(36);index:idx_files_creator_id_created_at,priority:1"`
	Hash      string                 `gorm:"type:char(32);not null"`
	// default:'' deleted
	Type            model.FileType         `gorm:"type:varchar(30);not null"`
	IsAnimatedImage bool                   `gorm:"type:boolean;not null;default:false"`
	ChannelID       optional.Of[uuid.UUID] `gorm:"type:char(36);index:idx_files_channel_id_created_at,priority:1"`
	CreatedAt       time.Time              `gorm:"precision:6;index:idx_files_channel_id_created_at,priority:2;index:idx_files_creator_id_created_at,priority:2"`
	DeletedAt       gorm.DeletedAt         `gorm:"precision:6"`
}

func (*v27FileMeta) TableName() string {
	return "files"
}

type v27FileThumbnail struct {
	FileID uuid.UUID           `gorm:"type:char(36);not null;primaryKey"`
	Type   model.ThumbnailType `gorm:"type:varchar(30);not null;primaryKey"`
	Mime   string              `gorm:"type:text;not null"`
	// int(11) -> bigint(20)
	Width int `gorm:"type:int;not null;default:0"`
	// int(11) -> bigint(20)
	Height int `gorm:"type:int;not null;default:0"`
}

func (*v27FileThumbnail) TableName() string {
	return "files_thumbnails"
}

type v27MessageStamp struct {
	MessageID uuid.UUID `gorm:"type:char(36);not null;primaryKey;index" json:"-"`
	StampID   uuid.UUID `gorm:"type:char(36);not null;primaryKey;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:2" json:"stampId"`
	UserID    uuid.UUID `gorm:"type:char(36);not null;primaryKey;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:1" json:"userId"`
	// int(11) -> bigint(20)
	Count     int       `gorm:"type:int;not null" json:"count"`
	CreatedAt time.Time `gorm:"precision:6" json:"createdAt"`
	UpdatedAt time.Time `gorm:"precision:6;index;index:idx_messages_stamps_user_id_stamp_id_updated_at,priority:3" json:"updatedAt"`
}

func (*v27MessageStamp) TableName() string {
	return "messages_stamps"
}

type v27OAuth2Authorize struct {
	Code     string    `gorm:"type:varchar(36);primaryKey"`
	ClientID string    `gorm:"type:char(36)"`
	UserID   uuid.UUID `gorm:"type:char(36)"`
	// int(11) -> bigint(20)
	ExpiresIn           int
	RedirectURI         string             `gorm:"type:text"`
	Scopes              model.AccessScopes `gorm:"type:text"`
	OriginalScopes      model.AccessScopes `gorm:"type:text"`
	CodeChallenge       string             `gorm:"type:varchar(128)"`
	CodeChallengeMethod string             `gorm:"type:text"`
	Nonce               string             `gorm:"type:text"`
	CreatedAt           time.Time          `gorm:"precision:6"`
}

func (*v27OAuth2Authorize) TableName() string {
	return "oauth2_authorizes"
}

type v27OAuth2Token struct {
	ID             uuid.UUID          `gorm:"type:char(36);primaryKey"`
	ClientID       string             `gorm:"type:char(36)"`
	UserID         uuid.UUID          `gorm:"type:char(36)"`
	RedirectURI    string             `gorm:"type:text"`
	AccessToken    string             `gorm:"type:varchar(36);unique"`
	RefreshToken   string             `gorm:"type:varchar(36);unique"`
	RefreshEnabled bool               `gorm:"type:boolean;default:false"`
	Scopes         model.AccessScopes `gorm:"type:text"`
	// int(11) -> bigint(20)
	ExpiresIn int
	CreatedAt time.Time      `gorm:"precision:6"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"`
}

func (*v27OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

type v27OgpCache struct {
	// int(11) -> bigint(20)
	ID        int       `gorm:"auto_increment;not null;primaryKey"`
	URL       string    `gorm:"type:text;not null"`
	URLHash   string    `gorm:"type:char(40);not null;index"`
	Valid     bool      `gorm:"type:boolean"`
	Content   model.Ogp `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"precision:6"`
}

func (*v27OgpCache) TableName() string {
	return "ogp_cache"
}
