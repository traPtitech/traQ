package ogp

import "errors"

var (
	// ErrParse 対象URLをHTMLとしてパースできませんでした
	ErrParse = errors.New("parse error")
	// ErrNetwork 対象URLにアクセスできませんでした
	ErrNetwork = errors.New("no such host")
	// ErrClient 対象URLにアクセスした際に4xxエラーが発生しました
	ErrClient = errors.New("network error (client)")
	// ErrServer 対象URLにアクセスした際に5xxエラーが発生しました
	ErrServer = errors.New("network error (server)")
)
