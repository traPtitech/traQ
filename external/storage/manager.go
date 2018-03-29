package storage

import (
	"errors"
	"io"
)

var (
	// ErrUnknownManager : fileエラー 不明なファイルマネージャー
	ErrUnknownManager = errors.New("unknown file manager")
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

// FileManagers ファイルマネージャーのマップ
type FileManagers map[string]FileManager
