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

type ThumbnailType int

// Value database/sql/driver.Valuer 実装
func (t ThumbnailType) Value() (driver.Value, error) {
	v := t.String()
	if v == "null" {
		return nil, errors.New("unknown ThumbnailType")
	}
	return v, nil
}

// Scan database/sql.Scanner 実装
func (t *ThumbnailType) Scan(src interface{}) (err error) {
	switch s := src.(type) {
	case string:
		*t, err = ThumbnailTypeFromString(s)
	case []byte:
		*t, err = ThumbnailTypeFromString(string(s))
	default:
		err = errors.New("failed to scan ThumbnailType")
	}
	return
}

func (t ThumbnailType) String() string {
	switch t {
	case ThumbnailTypeImage:
		return "image"
	case ThumbnailTypeWaveform:
		return "waveform"
	default:
		return "null"
	}
}

// Suffix storageに収納する際のkey suffix
func (t ThumbnailType) Suffix() string {
	switch t {
	case ThumbnailTypeImage:
		return "thumb"
	case ThumbnailTypeWaveform:
		return "waveform"
	default:
		return "null"
	}
}

func ThumbnailTypeFromString(s string) (ThumbnailType, error) {
	switch strings.ToLower(s) {
	case "image":
		return ThumbnailTypeImage, nil
	case "waveform":
		return ThumbnailTypeWaveform, nil
	default:
		return 0, errors.New("unknown ThumbnailType")
	}
}

const (
	// ThumbnailTypeImage 通常サムネイル画像
	ThumbnailTypeImage ThumbnailType = iota + 1
	// ThumbnailTypeWaveform 波形画像
	ThumbnailTypeWaveform
)

type File interface {
	GetID() uuid.UUID
	GetFileName() string
	GetMIMEType() string
	GetFileSize() int64
	GetFileType() FileType
	GetCreatorID() optional.UUID
	GetMD5Hash() string
	IsAnimatedImage() bool
	GetUploadChannelID() optional.UUID
	GetCreatedAt() time.Time
	GetThumbnail(thumbnailType ThumbnailType) (bool, FileThumbnail)

	Open() (ioext.ReadSeekCloser, error)
	OpenThumbnail(thumbnailType ThumbnailType) (ioext.ReadSeekCloser, error)
	GetAlternativeURL() string
}

// FileMeta DBに格納するファイルの構造体
type FileMeta struct {
	ID              uuid.UUID       `gorm:"type:char(36);not null;primary_key"`
	Name            string          `gorm:"type:text;not null"`
	Mime            string          `gorm:"type:text;not null"`
	Size            int64           `gorm:"type:bigint;not null"`
	CreatorID       optional.UUID   `gorm:"type:char(36)"`
	Hash            string          `gorm:"type:char(32);not null"`
	Type            FileType        `gorm:"type:varchar(30);not null;default:''"`
	IsAnimatedImage bool            `gorm:"type:boolean;not null;default:false"`
	ChannelID       optional.UUID   `gorm:"type:char(36)"`
	CreatedAt       time.Time       `gorm:"precision:6"`
	DeletedAt       *time.Time      `gorm:"precision:6"`
	Thumbnails      []FileThumbnail `gorm:"association_autoupdate:false;association_autocreate:false;preload:false;foreignKey:FileID"`
}

// TableName dbのtableの名前を返します
func (f FileMeta) TableName() string {
	return "files"
}

// FileThumbnail ファイルのサムネイル情報の構造体
type FileThumbnail struct {
	FileID uuid.UUID     `gorm:"type:char(36);not null;primary_key"`
	Type   ThumbnailType `gorm:"type:varchar(30);not null;primary_key"`
	Mime   string        `gorm:"type:text;not null"`
	Width  int           `gorm:"type:int;not null;default:0"`
	Height int           `gorm:"type:int;not null;default:0"`
}

func (f FileThumbnail) TableName() string {
	return "files_thumbnails"
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
