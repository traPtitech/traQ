package model

import (
	"database/sql"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/ioext"
	"gopkg.in/guregu/null.v3"
	"time"
)

const (
	// FileTypeUserFile ユーザーアップロードファイルタイプ
	FileTypeUserFile = ""
	// FileTypeIcon ユーザーアイコンファイルタイプ
	FileTypeIcon = "icon"
	// FileTypeStamp スタンプファイルタイプ
	FileTypeStamp = "stamp"
	// FileTypeThumbnail サムネイルファイルタイプ
	FileTypeThumbnail = "thumbnail"
)

type FileMeta interface {
	GetID() uuid.UUID
	GetFileName() string
	GetMIMEType() string
	GetFileSize() int64
	GetFileType() string
	GetCreatorID() uuid.NullUUID
	GetMD5Hash() string
	HasThumbnail() bool
	GetThumbnailMIMEType() string
	GetThumbnailWidth() int
	GetThumbnailHeight() int
	GetUploadChannelID() uuid.NullUUID
	GetCreatedAt() time.Time

	Open() (ioext.ReadSeekCloser, error)
	OpenThumbnail() (ioext.ReadSeekCloser, error)
	GetAlternativeURL() string
}

// File DBに格納するファイルの構造体
type File struct {
	ID              uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Name            string        `gorm:"type:text;not null"`
	Mime            string        `gorm:"type:text;not null"`
	Size            int64         `gorm:"type:bigint;not null"`
	CreatorID       uuid.NullUUID `gorm:"type:char(36)"`
	Hash            string        `gorm:"type:char(32);not null"`
	Type            string        `gorm:"type:varchar(30);not null;default:''"`
	HasThumbnail    bool          `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   null.String   `gorm:"type:text"`
	ThumbnailWidth  int           `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int           `gorm:"type:int;not null;default:0"`
	ChannelID       uuid.NullUUID `gorm:"type:char(36)"`
	CreatedAt       time.Time     `gorm:"precision:6"`
	DeletedAt       *time.Time    `gorm:"precision:6"`
}

// TableName dbのtableの名前を返します
func (f File) TableName() string {
	return "files"
}

// FileACLEntry ファイルアクセスコントロールリストエントリー構造体
type FileACLEntry struct {
	FileID uuid.UUID     `gorm:"type:char(36);primary_key;not null"`
	UserID uuid.NullUUID `gorm:"type:char(36);primary_key;not null"`
	Allow  sql.NullBool  `gorm:"not null"`
}

// TableName FileACLEntry構造体のテーブル名
func (f FileACLEntry) TableName() string {
	return "files_acl"
}
