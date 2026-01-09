package parser

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
	// ErrDomainRequest 特殊処理を行うドメインのURLが期待した形式ではありませんでした
	ErrDomainRequest = errors.New("bad request for special domain ")
	// ErrSSRF はSSRF攻撃の可能性がある場合に返されるエラーです
	ErrSSRF = errors.New("SSRF detected: access to internal network is not allowed")
)
