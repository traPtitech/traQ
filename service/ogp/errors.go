package ogp

import "errors"

var (
	// ErrParse 対象URLをHTMLとしてパースできませんでした
	ErrParse = errors.New("parse error")
	// ErrNetwork 対象URLにアクセスできませんでした
	ErrNetwork = errors.New("network error")
	// ErrContentTypeNotSupported 対象URLのコンテンツがサポートされていないContent-Typeを持っていました
	ErrContentTypeNotSupported = errors.New("content type not supported")
	// ErrClient 対象URLにアクセスした際に4xxエラーが発生しました
	ErrClient = errors.New("network error (client)")
	// ErrServer 対象URLにアクセスした際に5xxエラーが発生しました
	ErrServer = errors.New("network error (server)")
)
