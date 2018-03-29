package storage

import (
	"fmt"
	"io"
	"os"
)

// LocalFileManager ローカルファイルマネージャー
type LocalFileManager struct {
	dirName string
}

// NewLocalFileManager LocalFileManagerのコンストラクタ
func NewLocalFileManager() *LocalFileManager {
	fm := &LocalFileManager{}
	if dir := os.Getenv("TRAQ_TEMP"); dir != "" {
		fm.dirName = dir
	} else {
		fm.dirName = "../resources"
	}
	return fm
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
