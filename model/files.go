package model

import (
	"database/sql"
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/gofrs/uuid"
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

// File DBに格納するファイルの構造体
type File struct {
	ID              uuid.UUID  `gorm:"type:char(36);not null;primary_key"   json:"fileId"`
	Name            string     `gorm:"type:text;not null"                   json:"name"`
	Mime            string     `gorm:"type:text;not null"                   json:"mime"`
	Size            int64      `gorm:"type:bigint;not null"                 json:"size"`
	CreatorID       uuid.UUID  `gorm:"type:char(36);not null"               json:"-"`
	Hash            string     `gorm:"type:char(32);not null"               json:"md5"`
	Type            string     `gorm:"type:varchar(30);not null;default:''" json:"-"`
	HasThumbnail    bool       `gorm:"type:boolean;not null;default:false"  json:"hasThumb"`
	ThumbnailWidth  int        `gorm:"type:int;not null;default:0"          json:"thumbWidth,omitempty"`
	ThumbnailHeight int        `gorm:"type:int;not null;default:0"          json:"thumbHeight,omitempty"`
	CreatedAt       time.Time  `gorm:"precision:6"                          json:"datetime"`
	DeletedAt       *time.Time `gorm:"precision:6"                          json:"-"`
}

// TableName dbのtableの名前を返します
func (f *File) TableName() string {
	return "files"
}

// Validate 構造体を検証します
func (f File) Validate() error {
	return vd.ValidateStruct(&f,
		vd.Field(&f.Name, vd.Required),
		vd.Field(&f.Mime, vd.Required),
		vd.Field(&f.Size, vd.Min(0)),
		vd.Field(&f.Hash, vd.Required, vd.Length(32, 32), is.Alphanumeric),
		vd.Field(&f.ThumbnailWidth, vd.Min(0)),
		vd.Field(&f.ThumbnailHeight, vd.Min(0)),
	)
}

// GetKey ファイルのストレージに対するキーを返す
func (f *File) GetKey() string {
	return f.ID.String()
}

// GetThumbKey ファイルのサムネイルのストレージに対するキーを返す
func (f *File) GetThumbKey() string {
	return f.ID.String() + "-thumb"
}

// FileACLEntry ファイルアクセスコントロールリストエントリー構造体
type FileACLEntry struct {
	FileID uuid.UUID     `gorm:"type:char(36);primary_key;not null"`
	UserID uuid.NullUUID `gorm:"type:char(36);primary_key;not null"`
	Allow  sql.NullBool  `gorm:"not null"`
}

// TableName FileACLEntry構造体のテーブル名
func (f *FileACLEntry) TableName() string {
	return "files_acl"
}
