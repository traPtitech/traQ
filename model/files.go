package model

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

var fileManagers = map[string]FileManager{
	"": NewDevFileManager(),
}

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
	ID        string    `xorm:"char(36) pk"`
	Name      string    `xorm:"text not null"`
	Mime      string    `xorm:"text not null"`
	Size      int64     `xorm:"bigint not null"`
	CreatorID string    `xorm:"char(36) not null"`
	IsDeleted bool      `xorm:"bool not null"`
	Hash      string    `xorm:"char(32) not null"`
	Manager   string    `xorm:"varchar(30) not null default ''"`
	CreatedAt time.Time `xorm:"created not null"`
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
		return fmt.Errorf("unknown file manager: %s", f.Manager)
	}

	if err := writer.WriteByID(src, f.ID, f.Name, f.Mime); err != nil {
		return fmt.Errorf("Failed to write data into file: %v", err)
	}

	hash, err := calcMD5(src)
	if err != nil {
		return err
	}
	f.Hash = hash

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
		return nil, fmt.Errorf("unknown file manager: %s", f.Manager)
	}

	return reader.OpenFileByID(f.ID)
}

// GetRedirectURL リダイレクト先URLが存在する場合はそれを返します
func (f *File) GetRedirectURL() string {
	m, ok := fileManagers[f.Manager]
	if !ok {
		return ""
	}
	return m.GetRedirectURL(f.ID)
}

// OpenFileByID ファイルを取得します
func OpenFileByID(ID string) (io.ReadCloser, error) {
	meta, err := GetMetaFileDataByID(ID)
	if err != nil {
		return nil, err
	}

	reader, ok := fileManagers[meta.Manager]
	if !ok {
		return nil, fmt.Errorf("unknown file manager: %s", meta.Manager)
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

// 与えられたデータに対してMD5によるハッシュ値を計算します
func calcMD5(src io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", fmt.Errorf("Failed to calc md5")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
