package storage

import (
	"errors"
	"io"
)

var (
	// ErrFileNotFound 指定されたキーのファイルは見つかりません
	ErrFileNotFound = errors.New("file not found")
)

// FileStorage ファイルストレージのインターフェース
type FileStorage interface {
	// SaveByKey srcをkeyのファイルとして保存する
	SaveByKey(src io.Reader, key, name, contentType, fileType string) error
	// OpenFileByKey keyで指定されたファイルを読み込む
	OpenFileByKey(key string) (ReadSeekCloser, error)
	// DeleteByKey keyで指定されたファイルを削除する
	DeleteByKey(key string) error
	// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。発行機能がない場合は空文字列を返します(エラーはありません)。
	GenerateAccessURL(key string) (string, error)
}

// ReadSeekCloser io.Reader, io.Closer, io.Seekerの複合インターフェイス
type ReadSeekCloser interface {
	io.Reader
	io.Closer
	io.Seeker
}
