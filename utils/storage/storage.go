package storage

import (
	"errors"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/ioext"
	"io"
)

var (
	// ErrFileNotFound 指定されたキーのファイルは見つかりません
	ErrFileNotFound = errors.New("file not found")
)

// FileStorage ファイルストレージのインターフェース
type FileStorage interface {
	// SaveByKey srcをkeyのファイルとして保存する
	SaveByKey(src io.Reader, key, name, contentType string, fileType model.FileType) error
	// OpenFileByKey keyで指定されたファイルを読み込む
	OpenFileByKey(key string, fileType model.FileType) (ioext.ReadSeekCloser, error)
	// DeleteByKey keyで指定されたファイルを削除する
	DeleteByKey(key string, fileType model.FileType) error
	// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。発行機能がない場合は空文字列を返します(エラーはありません)。
	GenerateAccessURL(key string, fileType model.FileType) (string, error)
}
