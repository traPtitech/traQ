package repository

import "errors"

var (
	// ErrNilID 汎用エラー 引数のIDがNilです
	ErrNilID = errors.New("nil id")
	// ErrNotFound 汎用エラー 見つかりません
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists 汎用エラー 既に存在しています
	ErrAlreadyExists = errors.New("already exists")
	// ErrForbidden 汎用エラー 禁止されています
	ErrForbidden = errors.New("forbidden")
)
