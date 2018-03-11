package model

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/utils"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

const (
	thumbnailMaxWidth  = 360
	thumbnailMaxHeight = 480
	thumbnailRatio     = float64(thumbnailMaxWidth) / float64(thumbnailMaxHeight)
)

var (
	fileManagers = map[string]FileManager{
		"": NewDevFileManager(),
	}

	// ErrFileThumbUnsupported : fileエラー この形式のファイルのサムネイル生成はサポートされていない
	ErrFileThumbUnsupported = errors.New("generating a thumbnail of the file is not supported")
	// ErrFileUnknownManager : fileエラー 不明なファイルマネージャー
	ErrFileUnknownManager = errors.New("unknown file manager")
)

// FileManager ファイルを読み書きするマネージャーのインターフェース
type FileManager interface {
	// srcをIDのファイルとして保存する
	WriteByID(src io.Reader, ID, name, contentType string) error
	// IDで指定されたファイルを読み込む
	OpenFileByID(ID string) (io.ReadCloser, error)
	// IDで指定されたファイルを削除する
	DeleteByID(ID string) error
	// RedirectURLが発行できる場合は取得します。出来ない場合は空文字列を返します
	GetRedirectURL(ID string) string
}

// File DBに格納するファイルの構造体
type File struct {
	ID              string    `xorm:"char(36) pk"`
	Name            string    `xorm:"text not null"`
	Mime            string    `xorm:"text not null"`
	Size            int64     `xorm:"bigint not null"`
	CreatorID       string    `xorm:"char(36) not null"`
	IsDeleted       bool      `xorm:"bool not null"`
	Hash            string    `xorm:"char(32) not null"`
	Manager         string    `xorm:"varchar(30) not null default ''"`
	HasThumbnail    bool      `xorm:"bool not null"`
	ThumbnailWidth  int       `xorm:"int not null"`
	ThumbnailHeight int       `xorm:"int not null"`
	CreatedAt       time.Time `xorm:"created not null"`
}

// TableName dbのtableの名前を返します
func (f *File) TableName() string {
	return "files"
}

// Create file構造体を作ります
func (f *File) Create(src io.Reader) error {
	if f.Name == "" {
		return fmt.Errorf("file name is empty")
	}
	if f.Size == 0 {
		return fmt.Errorf("file size is 0")
	}
	if f.CreatorID == "" {
		return fmt.Errorf("file creatorID is empty")
	}

	f.ID = CreateUUID()
	f.IsDeleted = false
	f.Mime = mime.TypeByExtension(filepath.Ext(f.Name))

	writer, ok := fileManagers[f.Manager]
	if !ok {
		return ErrFileUnknownManager
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
		return nil, ErrFileUnknownManager
	}

	return reader.OpenFileByID(f.ID)
}

// OpenThumbnail サムネイルファイルを開きます
func (f *File) OpenThumbnail() (io.ReadCloser, error) {
	reader, ok := fileManagers[f.Manager]
	if !ok {
		return nil, ErrFileUnknownManager
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

// GenerateThumbnail サムネイル画像を生成します
func GenerateThumbnail(ctx context.Context, f *File, src io.Reader) error {
	var (
		img       image.Image
		err       error
		thumbSize image.Point
		dst       draw.Image
		b         = &bytes.Buffer{}
	)

	writer, ok := fileManagers[f.Manager]
	if !ok {
		return ErrFileUnknownManager
	}

	switch f.Mime {
	case "image/png":
		img, err = png.Decode(src)
	case "image/gif":
		img, err = gif.Decode(src)
	case "image/jpeg":
		img, err = jpeg.Decode(src)
	case "image/bmp":
		img, err = bmp.Decode(src)
	case "image/webp":
		img, err = webp.Decode(src)
	default: // Unsupported Type
		return ErrFileThumbUnsupported
	}
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		thumbSize = calcThumbnailSize(img.Bounds().Size())
		dst = image.NewRGBA(image.Rectangle{Min: image.ZP, Max: thumbSize})
		draw.Draw(dst, dst.Bounds(), image.White, image.ZP, draw.Src)
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if err := jpeg.Encode(b, dst, &jpeg.Options{Quality: 100}); err != nil {
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if err := writer.WriteByID(b, f.ID+"-thumb", f.ID+"-thumb.jpg", "image/jpeg"); err != nil {
			return err
		}
	}

	f.HasThumbnail = true
	f.ThumbnailWidth = thumbSize.X
	f.ThumbnailHeight = thumbSize.Y

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
		return nil, ErrFileUnknownManager
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

// サムネイルのサイズを計算します
func calcThumbnailSize(size image.Point) image.Point {
	if size.X <= thumbnailMaxWidth && size.Y <= thumbnailMaxHeight {
		// 元画像がサムネイル画像より小さい
		return size
	}

	ratio := float64(size.X) / float64(size.Y)

	if ratio > thumbnailRatio {
		return image.Pt(thumbnailMaxWidth, int(thumbnailMaxWidth/ratio))
	}
	return image.Pt(int(thumbnailMaxHeight*ratio), thumbnailMaxHeight)
}

// SetFileManager ファイルマネージャーリストにマネージャーをセットします
func SetFileManager(name string, manager FileManager) {
	fileManagers[name] = manager
}

// 以下、開発環境用

// LocalFileManager 開発用。routerの方でも使用するために公開
type LocalFileManager struct {
	dirName string
}

// OpenFileByID ファイルを取得します
func (fm *LocalFileManager) OpenFileByID(ID string) (io.ReadCloser, error) {
	fileName := fm.dirName + "/" + ID
	if _, err := os.Stat(fileName); err != nil {
		return nil, fmt.Errorf("Invalid ID: %s", ID)
	}

	reader, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %v", err)
	}

	return reader, nil
}

// WriteByID srcの内容をIDで指定されたファイルに書き込みます
func (fm *LocalFileManager) WriteByID(src io.Reader, ID, name, contentType string) error {
	if _, err := os.Stat(fm.dirName); err != nil {
		if err = os.Mkdir(fm.dirName, 0700); err != nil {
			return fmt.Errorf("Can't create directory: %v", err)
		}
	}

	file, err := os.Create(fm.dirName + "/" + ID)
	if err != nil {
		return fmt.Errorf("Failed to open file: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, src); err != nil {
		return fmt.Errorf("Failed to write into file %v", err)
	}
	return nil
}

// DeleteByID ファイルを削除します
func (fm *LocalFileManager) DeleteByID(ID string) error {
	fileName := fm.dirName + "/" + ID
	if _, err := os.Stat(fileName); err != nil {
		return err
	}
	return os.Remove(fileName)
}

// GetRedirectURL 必ず空文字列を返します
func (*LocalFileManager) GetRedirectURL(ID string) string {
	return ""
}

// GetDir ファイルの保存先を取得する
func (fm *LocalFileManager) GetDir() string {
	return fm.dirName
}

// NewDevFileManager DevFileManagerのコンストラクタ
func NewDevFileManager() *LocalFileManager {
	fm := &LocalFileManager{}
	if dir := os.Getenv("TRAQ_TEMP"); dir != "" {
		fm.dirName = dir
	} else {
		fm.dirName = "../resources"
	}
	return fm
}
