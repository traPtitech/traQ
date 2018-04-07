package model

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
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
	ID              string    `xorm:"char(36) pk"                    validate:"uuid,required"`
	Name            string    `xorm:"text not null"                  validate:"required"`
	Mime            string    `xorm:"text not null"                  validate:"required"`
	Size            int64     `xorm:"bigint not null"                validate:"min=0,required"`
	CreatorID       string    `xorm:"char(36) not null"              validate:"uuid,required"`
	IsDeleted       bool      `xorm:"bool not null"`
	Hash            string    `xorm:"char(32) not null"              validate:"max=32"`
	Manager         string    `xorm:"varchar(30) not null default ''"`
	HasThumbnail    bool      `xorm:"bool not null"`
	ThumbnailWidth  int       `xorm:"int not null"                   validate:"min=0"`
	ThumbnailHeight int       `xorm:"int not null"                   validate:"min=0"`
	CreatedAt       time.Time `xorm:"created not null"`
}

// TableName dbのtableの名前を返します
func (f *File) TableName() string {
	return "files"
}

// Validate 構造体を検証します
func (f *File) Validate() error {
	return validator.ValidateStruct(f)
}

// Create file構造体を作ります
func (f *File) Create(src io.Reader) error {
	f.ID = CreateUUID()
	f.IsDeleted = false
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
			return fmt.Errorf("Failed to write data into file: %v", err)
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

	if _, err := db.Insert(f); err != nil {
		return fmt.Errorf("Failed to create file")
	}
	return nil
}

// Exists ファイルが存在するかを判定します
func (f *File) Exists() (bool, error) {
	if f.ID == "" {
		return false, fmt.Errorf("file ID is empty")
	}
	return db.Get(f)
}

// Delete file構造体をDBから消去します
func (f *File) Delete() error {
	f.IsDeleted = true
	if _, err := db.ID(f.ID).UseBool().Update(f); err != nil {
		return err
	}

	m, ok := fileManagers[f.Manager]
	if ok {
		return m.DeleteByID(f.ID)
	}

	return nil
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

	if _, err := db.ID(f.ID).UseBool().MustCols().Update(f); err != nil {
		return err
	}
	return nil
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
func OpenFileByID(ID string) (io.ReadCloser, error) {
	meta, err := GetMetaFileDataByID(ID)
	if err != nil {
		return nil, err
	}

	reader, ok := fileManagers[meta.Manager]
	if !ok {
		return nil, storage.ErrUnknownManager
	}

	return reader.OpenFileByID(ID)
}

// GetMetaFileDataByID ファイルのメタデータを取得します
func GetMetaFileDataByID(FileID string) (*File, error) {
	f := &File{}

	has, err := db.ID(FileID).Get(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to find file")
	}
	if !has {
		return nil, nil
	}

	return f, nil
}

// SetFileManager ファイルマネージャーリストにマネージャーをセットします
func SetFileManager(name string, manager storage.FileManager) {
	fileManagers[name] = manager
}
