package model

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/external/storage"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/thumb"
	"github.com/traPtitech/traQ/utils/validator"
	"golang.org/x/sync/errgroup"
	"io"
	"mime"
	"path/filepath"
	"time"
)

var (
	fileManagers = storage.FileManagers{
		"": storage.NewLocalFileManager(config.LocalStorageDir),
	}

	// ErrFileThumbUnsupported : fileエラー この形式のファイルのサムネイル生成はサポートされていない
	ErrFileThumbUnsupported = errors.New("generating a thumbnail of the file is not supported")
)

// File DBに格納するファイルの構造体
type File struct {
	ID              string     `gorm:"type:char(36);primary_key" json:"fileId"   validate:"uuid,required"`
	Name            string     `gorm:"type:text"                 json:"name"     validate:"required"`
	Mime            string     `gorm:"type:text"                 json:"mime"     validate:"required"`
	Size            int64      `                                 json:"size"     validate:"min=0,required"`
	CreatorID       string     `gorm:"type:char(36)"             json:"-"        validate:"uuid,required"`
	Hash            string     `gorm:"type:char(32)"             json:"md5"      validate:"max=32"`
	Manager         string     `gorm:"type:varchar(30)"          json:"-"`
	HasThumbnail    bool       `                                 json:"hasThumb"`
	ThumbnailWidth  int        `                                 json:"thumbWidth,omitempty"  validate:"min=0"`
	ThumbnailHeight int        `                                 json:"thumbHeight,omitempty" validate:"min=0"`
	CreatedAt       time.Time  `gorm:"precision:6"               json:"datetime"`
	DeletedAt       *time.Time `gorm:"precision:6"               json:"-"`
}

// GetID FileのUUIDを返します
func (f *File) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(f.ID))
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
	m, ok := fileManagers[f.Manager]
	if ok {
		if f.HasThumbnail {
			if err := m.DeleteByID(f.ID + "-thumb"); err != nil {
				return err
			}
		}
		return m.DeleteByID(f.ID)
	}
	return nil
}

// Validate 構造体を検証します
func (f *File) Validate() error {
	return validator.ValidateStruct(f)
}

// Create file構造体を作ります
func (f *File) Create(src io.Reader) error {
	f.ID = CreateUUID()
	f.Mime = mime.TypeByExtension(filepath.Ext(f.Name))
	if len(f.CreatorID) == 0 {
		f.CreatorID = serverUser.ID
	}

	writer, ok := fileManagers[f.Manager]
	if !ok {
		return storage.ErrUnknownManager
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
		io.Copy(utils.MultiWriter(fileWriter, hash, thumbWriter), src) // 並列化してるけど、pipeじゃなくてbuffer使わないとpipeがブロックしてて意味無い疑惑
	}()

	// fileの保存
	eg.Go(func() error {
		defer fileSrc.Close()
		if err := writer.WriteByID(fileSrc, f.ID, f.Name, f.Mime); err != nil {
			return err
		}
		return nil
	})

	// サムネイルの生成
	eg.Go(func() error {
		// アップロードされたファイルの拡張子が間違えてたり、変なの送ってきた場合
		// サムネイルを生成しないだけで全体のエラーにはしない
		defer thumbSrc.Close()
		if err := GenerateThumbnail(ctx, f, thumbSrc); err != nil {
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

// DeleteFile ファイルを削除します
func DeleteFile(fileID uuid.UUID) error {
	f, err := GetMetaFileDataByID(fileID)
	if err != nil {
		return err
	}

	return db.Delete(f).Error
}

// Open fileを開きます
func (f *File) Open() (io.ReadCloser, error) {
	reader, ok := fileManagers[f.Manager]
	if !ok {
		return nil, storage.ErrUnknownManager
	}

	return reader.OpenFileByID(f.ID)
}

// OpenThumbnail サムネイルファイルを開きます
func (f *File) OpenThumbnail() (io.ReadCloser, error) {
	reader, ok := fileManagers[f.Manager]
	if !ok {
		return nil, storage.ErrUnknownManager
	}

	return reader.OpenFileByID(f.ID + "-thumb")
}

// GetRedirectURL リダイレクト先URLが存在する場合はそれを返します
func (f *File) GetRedirectURL() string {
	m, ok := fileManagers[f.Manager]
	if !ok {
		return ""
	}
	return m.GetRedirectURL(f.ID)
}

// RegenerateThumbnail サムネイル画像を再生成します
func (f *File) RegenerateThumbnail() error {
	reader, ok := fileManagers[f.Manager]
	if !ok {
		return storage.ErrUnknownManager
	}

	//既存のものを削除
	reader.DeleteByID(f.ID + "-thumb")

	src, err := reader.OpenFileByID(f.ID)
	if err != nil {
		return err
	}

	if err := GenerateThumbnail(context.Background(), f, src); err != nil {
		return err
	}

	return db.Model(f).Updates(map[string]interface{}{
		"has_thumbnail":    true,
		"thumbnail_width":  f.ThumbnailWidth,
		"thumbnail_height": f.ThumbnailHeight,
	}).Error
}

// GenerateThumbnail サムネイル画像を生成します
func GenerateThumbnail(ctx context.Context, f *File, src io.Reader) error {
	writer, ok := fileManagers[f.Manager]
	if !ok {
		return storage.ErrUnknownManager
	}

	img, err := thumb.Generate(ctx, src, f.Mime)
	if err != nil {
		return err
	}

	b := &bytes.Buffer{}
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
		if err := writer.WriteByID(b, f.ID+"-thumb", f.ID+"-thumb.png", "image/png"); err != nil {
			return err
		}
	}

	f.HasThumbnail = true
	f.ThumbnailWidth = img.Bounds().Size().X
	f.ThumbnailHeight = img.Bounds().Size().Y

	return nil
}

// OpenFileByID ファイルを取得します
func OpenFileByID(fileID uuid.UUID) (io.ReadCloser, error) {
	meta, err := GetMetaFileDataByID(fileID)
	if err != nil {
		return nil, err
	}

	reader, ok := fileManagers[meta.Manager]
	if !ok {
		return nil, storage.ErrUnknownManager
	}

	return reader.OpenFileByID(fileID.String())
}

// GetMetaFileDataByID ファイルのメタデータを取得します
func GetMetaFileDataByID(fileID uuid.UUID) (*File, error) {
	f := &File{}
	if err := db.Where(&File{ID: fileID.String()}).Take(f).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return f, nil
}

// SetFileManager ファイルマネージャーリストにマネージャーをセットします
func SetFileManager(name string, manager storage.FileManager) {
	fileManagers[name] = manager
}
