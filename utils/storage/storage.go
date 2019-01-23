package storage

import (
	"errors"
	"io"
)

// ErrFileNotFound 指定されたキーのファイルは見つかりません
var ErrFileNotFound = errors.New("file not found")

// FileStorage ファイルストレージのインターフェース
type FileStorage interface {
	// SaveByKey srcをkeyのファイルとして保存する
	SaveByKey(src io.Reader, key, name, contentType string) error
	// OpenFileByKey keyで指定されたファイルを読み込む
	OpenFileByKey(key string) (io.ReadCloser, error)
	// DeleteByKey keyで指定されたファイルを削除する
	DeleteByKey(key string) error
}
