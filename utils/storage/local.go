package storage

import (
	"io"
	"os"
)

// LocalFileStorage ローカルファイルストレージ
type LocalFileStorage struct {
	dirName string
}

// NewLocalFileStorage LocalFileStorageを生成します。指定したディレクトリは既に存在していなければいけません。
func NewLocalFileStorage(dir string) *LocalFileStorage {
	fs := &LocalFileStorage{}
	if dir != "" {
		fs.dirName = dir
	} else {
		fs.dirName = "./storage"
	}
	return fs
}

// OpenFileByKey ファイルを取得します
func (fs *LocalFileStorage) OpenFileByKey(key string) (ReadSeekCloser, error) {
	fileName := fs.getFilePath(key)
	reader, err := os.Open(fileName)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return reader, nil
}

// SaveByKey srcの内容をkeyで指定されたファイルに書き込みます
func (fs *LocalFileStorage) SaveByKey(src io.Reader, key, name, contentType, fileType string) error {
	file, err := os.Create(fs.getFilePath(key))
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, src); err != nil {
		return err
	}
	return nil
}

// DeleteByKey ファイルを削除します
func (fs *LocalFileStorage) DeleteByKey(key string) error {
	fileName := fs.getFilePath(key)
	if _, err := os.Stat(fileName); err != nil {
		return ErrFileNotFound
	}
	return os.Remove(fileName)
}

// GenerateAccessURL "",nilを返します
func (fs *LocalFileStorage) GenerateAccessURL(key string) (string, error) {
	return "", nil
}

// GetDir ファイルの保存先を取得する
func (fs *LocalFileStorage) GetDir() string {
	return fs.dirName
}

func (fs *LocalFileStorage) getFilePath(key string) string {
	return fs.dirName + "/" + key
}
