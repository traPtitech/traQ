package model

import (
	"database/sql/driver"
	"errors"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils/ioext"
	"github.com/traPtitech/traQ/utils/optional"
	"strings"
	"time"
)

type FileType int

// Value database/sql/driver.Valuer 実装
func (f FileType) Value() (driver.Value, error) {
	v := f.String()
	if v == "null" {
		return nil, errors.New("unknown FileType")
	}
	return v, nil
}

// Scan database/sql.Scanner 実装
func (f *FileType) Scan(src interface{}) (err error) {
	switch s := src.(type) {
	case string:
		*f, err = FileTypeFromString(s)
	case []byte:
		*f, err = FileTypeFromString(string(s))
	default:
		err = errors.New("failed to scan FileType")
	}
	return
}

func (f FileType) String() string {
	switch f {
	case FileTypeUserFile:
		return ""
	case FileTypeIcon:
		return "icon"
	case FileTypeStamp:
		return "stamp"
	case FileTypeThumbnail:
		return "thumbnail"
	default:
		return "null"
	}
}

func FileTypeFromString(s string) (FileType, error) {
	switch strings.ToLower(s) {
	case "":
		return FileTypeUserFile, nil
	case "icon":
		return FileTypeIcon, nil
	case "stamp":
		return FileTypeStamp, nil
	case "thumbnail":
		return FileTypeThumbnail, nil
	default:
		return 0, errors.New("unknown FileType")
	}
}

const (
	// FileTypeUserFile ユーザーアップロードファイルタイプ
	FileTypeUserFile FileType = iota
	// FileTypeIcon ユーザーアイコンファイルタイプ
	FileTypeIcon
	// FileTypeStamp スタンプファイルタイプ
	FileTypeStamp
	// FileTypeThumbnail サムネイルファイルタイプ
	FileTypeThumbnail
)

type FileMeta interface {
	GetID() uuid.UUID
	GetFileName() string
	GetMIMEType() string
	GetFileSize() int64
	GetFileType() FileType
	GetCreatorID() optional.UUID
	GetMD5Hash() string
	HasThumbnail() bool
	GetThumbnailMIMEType() string
	GetThumbnailWidth() int
	GetThumbnailHeight() int
	GetUploadChannelID() optional.UUID
	GetCreatedAt() time.Time

	Open() (ioext.ReadSeekCloser, error)
	OpenThumbnail() (ioext.ReadSeekCloser, error)
	GetAlternativeURL() string
}

// File DBに格納するファイルの構造体
type File struct {
	ID              uuid.UUID       `gorm:"type:char(36);not null;primary_key"`
	Name            string          `gorm:"type:text;not null"`
	Mime            string          `gorm:"type:text;not null"`
	Size            int64           `gorm:"type:bigint;not null"`
	CreatorID       optional.UUID   `gorm:"type:char(36)"`
	Hash            string          `gorm:"type:char(32);not null"`
	Type            FileType        `gorm:"type:varchar(30);not null;default:''"`
	HasThumbnail    bool            `gorm:"type:boolean;not null;default:false"`
	ThumbnailMime   optional.String `gorm:"type:text"`
	ThumbnailWidth  int             `gorm:"type:int;not null;default:0"`
	ThumbnailHeight int             `gorm:"type:int;not null;default:0"`
	ChannelID       optional.UUID   `gorm:"type:char(36)"`
	CreatedAt       time.Time       `gorm:"precision:6"`
	DeletedAt       *time.Time      `gorm:"precision:6"`
}

// TableName dbのtableの名前を返します
func (f File) TableName() string {
	return "files"
}

// FileACLEntry ファイルアクセスコントロールリストエントリー構造体
type FileACLEntry struct {
	FileID uuid.UUID     `gorm:"type:char(36);primary_key;not null"`
	UserID optional.UUID `gorm:"type:char(36);primary_key;not null"`
	Allow  optional.Bool `gorm:"not null"`
}

// TableName FileACLEntry構造体のテーブル名
func (f FileACLEntry) TableName() string {
	return "files_acl"
}
