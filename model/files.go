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

// FileWriter ファイルに書き込むインターフェース
type FileWriter interface {
	//*io.Readerのデータをstringで指定された名前で保存する
	WriteByID(io.Reader, string) error
}

// FileReader ファイルを読み込むインターフェース
type FileReader interface {
	//stringで指定された名前のファイルを取り出す
	OpenFileByID(string) (*os.File, error)
}

// File DBに格納するファイルの構造体
type File struct {
	ID        string    `xorm:"char(36) pk"`
	Name      string    `xorm:"text not null"`
	Mime      string    `xorm:"text not null"`
	Size      int64     `xomr:"bigint not null"`
	CreatorID string    `xorm:"char(36) not null"`
	IsDeleted bool      `xorm:"bool not null"`
	Hash      string    `xorm:"char(32) not null"`
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

	var writer FileWriter
	writer = &DevFileManager{} //dependent on dev environment
	if err := writer.WriteByID(src, f.ID); err != nil {
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

// Delete file構造体をDBから消去します
func (f *File) Delete() error {
	f.IsDeleted = true
	if _, err := db.ID(f.ID).UseBool().Update(f); err != nil {
		return fmt.Errorf("Failed to make Isdeleted true")
	}
	return nil
}

// OpenFileByID ファイルを取得します
func OpenFileByID(ID string) (*os.File, error) {
	//TODO: テストコード
	var reader FileReader
	reader = &DevFileManager{} //dependent on dev environment
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
		return nil, fmt.Errorf("The file doesn't exist")
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

// 以下、開発環境用

// DevFileManager 開発用。routerの方でも使用するために公開
type DevFileManager struct {
	dirName string
}

//OpenFileByID ファイルを取得します
func (fm *DevFileManager) OpenFileByID(ID string) (*os.File, error) {
	fm.setDir()
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
func (fm *DevFileManager) WriteByID(src io.Reader, ID string) error {
	fm.setDir()
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

// GetDir ファイルの保存先を取得する
func (fm *DevFileManager) GetDir() string {
	fm.setDir()
	return fm.dirName
}

func (fm *DevFileManager) setDir() {
	if fm.dirName == "" {
		if dir := os.Getenv("TRAQ_TEMP"); dir != "" {
			fm.dirName = dir
		} else {
			fm.dirName = "../resources"
		}
	}
}
