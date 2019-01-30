package model

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/storage"
	"github.com/traPtitech/traQ/utils/thumb"
	"github.com/traPtitech/traQ/utils/validator"
	"golang.org/x/sync/errgroup"
	"io"
	"mime"
	"path/filepath"
	"time"
)

const (
	// FileTypeUserFile ユーザーアップロードファイルタイプ
	FileTypeUserFile = ""
	// FileTypeIcon ユーザーアイコンファイルタイプ
	FileTypeIcon = "icon"
	// FileTypeStamp スタンプファイルタイプ
	FileTypeStamp = "stamp"
)

var (
	fs storage.FileStorage = storage.NewLocalFileStorage(config.LocalStorageDir)

	// ErrFileThumbUnsupported この形式のファイルのサムネイル生成はサポートされていない
	ErrFileThumbUnsupported = errors.New("generating a thumbnail of the file is not supported")
)

// File DBに格納するファイルの構造体
type File struct {
	ID              uuid.UUID  `gorm:"type:char(36);primary_key" json:"fileId"`
	Name            string     `gorm:"type:text"                 json:"name"     validate:"required"`
	Mime            string     `gorm:"type:text"                 json:"mime"     validate:"required"`
	Size            int64      `                                 json:"size"     validate:"min=0,required"`
	CreatorID       uuid.UUID  `gorm:"type:char(36)"             json:"-"`
	Hash            string     `gorm:"type:char(32)"             json:"md5"      validate:"max=32"`
	Type            string     `gorm:"type:varchar(30)"          json:"-"`
	HasThumbnail    bool       `                                 json:"hasThumb"`
	ThumbnailWidth  int        `                                 json:"thumbWidth,omitempty"  validate:"min=0"`
	ThumbnailHeight int        `                                 json:"thumbHeight,omitempty" validate:"min=0"`
	CreatedAt       time.Time  `gorm:"precision:6"               json:"datetime"`
	DeletedAt       *time.Time `gorm:"precision:6"               json:"-"`
}

// TableName dbのtableの名前を返します
func (f *File) TableName() string {
	return "files"
}

// BeforeDelete db.Deleteのトランザクション内で実行されます
func (f *File) BeforeDelete(scope *gorm.Scope) error {
	return db.Model(&File{ID: f.ID}).Take(f).Error
}

// AfterDelete db.Deleteのトランザクション内で実行されます
func (f *File) AfterDelete(scope *gorm.Scope) error {
	if f.HasThumbnail {
		if err := fs.DeleteByKey(f.getThumbKey()); err != nil {
			return err
		}
	}
	return fs.DeleteByKey(f.getKey())
}

// Validate 構造体を検証します
func (f *File) Validate() error {
	return validator.ValidateStruct(f)
}

// Create file構造体を作ります
func (f *File) Create(src io.Reader) error {
	f.ID = uuid.NewV4()
	f.Mime = mime.TypeByExtension(filepath.Ext(f.Name))
	if len(f.Mime) == 0 {
		f.Mime = echo.MIMEOctetStream
	}

	if err := f.Validate(); err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(context.Background())

	fileSrc, fileWriter := io.Pipe()
	thumbSrc, thumbWriter := io.Pipe()
	hash := md5.New()

	go func() {
		defer fileWriter.Close()
		defer thumbWriter.Close()
		_, _ = io.Copy(utils.MultiWriter(fileWriter, hash, thumbWriter), src) // 並列化してるけど、pipeじゃなくてbuffer使わないとpipeがブロックしてて意味無い疑惑
	}()

	// fileの保存
	eg.Go(func() error {
		defer fileSrc.Close()
		if err := fs.SaveByKey(fileSrc, f.getKey(), f.Name, f.Mime); err != nil {
			return err
		}
		return nil
	})

	// サムネイルの生成
	eg.Go(func() error {
		// アップロードされたファイルの拡張子が間違えてたり、変なの送ってきた場合
		// サムネイルを生成しないだけで全体のエラーにはしない
		defer thumbSrc.Close()
		if err := generateThumbnail(ctx, f, thumbSrc); err != nil {
			switch err {
			case ErrFileThumbUnsupported:
				return nil
			default:
				log.Error(err)
			}
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	f.Hash = hex.EncodeToString(hash.Sum(nil))

	return db.Create(f).Error
}

// Open fileを開きます
func (f *File) Open() (io.ReadCloser, error) {
	return fs.OpenFileByKey(f.getKey())
}

// OpenThumbnail サムネイルファイルを開きます
func (f *File) OpenThumbnail() (io.ReadCloser, error) {
	return fs.OpenFileByKey(f.getThumbKey())
}

// RegenerateThumbnail サムネイル画像を再生成します
func (f *File) RegenerateThumbnail() error {
	//既存のものを削除
	_ = fs.DeleteByKey(f.getThumbKey())

	src, err := fs.OpenFileByKey(f.getKey())
	if err != nil {
		return err
	}

	if err := generateThumbnail(context.Background(), f, src); err != nil {
		return err
	}

	return db.Model(f).Updates(map[string]interface{}{
		"has_thumbnail":    true,
		"thumbnail_width":  f.ThumbnailWidth,
		"thumbnail_height": f.ThumbnailHeight,
	}).Error
}

// getKey ファイルのストレージに対するキーを返す
func (f *File) getKey() string {
	return f.ID.String()
}

// getThumbKey ファイルのサムネイルのストレージに対するキーを返す
func (f *File) getThumbKey() string {
	return f.ID.String() + "-thumb"
}

// SaveFile ファイルを保存します。mimeが指定されていない場合はnameの拡張子によって決まります
func SaveFile(name string, src io.Reader, size int64, mime string, fType string) (uuid.UUID, error) {
	file := &File{
		Name: name,
		Size: size,
		Mime: mime,
		Type: fType,
	}
	if err := file.Create(src); err != nil {
		return uuid.Nil, err
	}

	return file.ID, nil
}

// DeleteFile ファイルを削除します
func DeleteFile(fileID uuid.UUID) error {
	f, err := GetMetaFileDataByID(fileID)
	if err != nil {
		return err
	}

	return db.Delete(f).Error
}

// OpenFileByID ファイルを取得します
func OpenFileByID(fileID uuid.UUID) (io.ReadCloser, error) {
	meta, err := GetMetaFileDataByID(fileID)
	if err != nil {
		return nil, err
	}
	return fs.OpenFileByKey(meta.getKey())
}

// GetMetaFileDataByID ファイルのメタデータを取得します
func GetMetaFileDataByID(fileID uuid.UUID) (*File, error) {
	if fileID == uuid.Nil {
		return nil, ErrNotFound
	}

	f := &File{}
	if err := db.Where(&File{ID: fileID}).Take(f).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// SetFileStorage ファイルストレージをセットします
func SetFileStorage(s storage.FileStorage) {
	fs = s
}

// generateThumbnail サムネイル画像を生成します
func generateThumbnail(ctx context.Context, f *File, src io.Reader) error {
	img, err := thumb.Generate(ctx, src, f.Mime)
	if err != nil {
		return err
	}

	var b *bytes.Buffer
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		b, err = thumb.EncodeToPNG(img)
		if err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if err := fs.SaveByKey(b, f.getThumbKey(), f.getThumbKey()+".png", "image/png"); err != nil {
			return err
		}
	}

	f.HasThumbnail = true
	f.ThumbnailWidth = img.Bounds().Size().X
	f.ThumbnailHeight = img.Bounds().Size().Y

	return nil
}
