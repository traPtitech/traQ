package ogp

import "errors"

var (
	// ErrNotFound 対象URLにアクセスしましたが、コンテンツが存在しませんでした
	ErrNotFound = errors.New("not found")
	// ErrClient 対象URLにアクセスした際に4xxエラーが発生しました
	ErrClient   = errors.New("network error (client)")
	// ErrServer 対象URLにアクセスした際に5xxエラーが発生しました
	ErrServer   = errors.New("network error (server)")
)