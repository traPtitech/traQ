package storage

import (
	"io"
	"os"

	"github.com/traPtitech/traQ/model"
)

// CompositeFileStorage 複合型ファイルストレージ
type CompositeFileStorage struct {
	swift *SwiftFileStorage
	local *LocalFileStorage
}

// NewCompositeFileStorage 引数の情報で複合型ファイルストレージを生成します
func NewCompositeFileStorage(localDir, container, userName, apiKey, tenant, tenantID, authURL, tempURLKey, cacheDir string) (*CompositeFileStorage, error) {
	l := NewLocalFileStorage(localDir)
	s, err := NewSwiftFileStorage(container, userName, apiKey, tenant, tenantID, authURL, tempURLKey, cacheDir)
	if err != nil {
		return nil, err
	}
	return &CompositeFileStorage{
		swift: s,
		local: l,
	}, nil
}

// SaveByKey srcをkeyのファイルとして保存する
func (fs *CompositeFileStorage) SaveByKey(src io.Reader, key, name, contentType string, fileType model.FileType) error {
	switch fileType {
	case model.FileTypeIcon, model.FileTypeStamp, model.FileTypeThumbnail:
		return fs.local.SaveByKey(src, key, name, contentType, fileType)
	default:
		return fs.swift.SaveByKey(src, key, name, contentType, fileType)
	}
}

// OpenFileByKey keyで指定されたファイルを読み込む
func (fs *CompositeFileStorage) OpenFileByKey(key string, fileType model.FileType) (io.ReadSeekCloser, error) {
	if _, err := os.Stat(fs.local.getFilePath(key)); os.IsNotExist(err) {
		return fs.swift.OpenFileByKey(key, fileType)
	}
	return fs.local.OpenFileByKey(key, fileType)
}

// DeleteByKey keyで指定されたファイルを削除する
func (fs *CompositeFileStorage) DeleteByKey(key string, fileType model.FileType) error {
	if _, err := os.Stat(fs.local.getFilePath(key)); os.IsNotExist(err) {
		return fs.swift.DeleteByKey(key, fileType)
	}
	return fs.local.DeleteByKey(key, fileType)
}

// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。発行機能がない場合は空文字列を返します(エラーはありません)。
func (fs *CompositeFileStorage) GenerateAccessURL(key string, fileType model.FileType) (string, error) {
	if _, err := os.Stat(fs.local.getFilePath(key)); os.IsNotExist(err) {
		return fs.swift.GenerateAccessURL(key, fileType)
	}
	return fs.local.GenerateAccessURL(key, fileType)
}
