package repository

import (
	"errors"
)

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

// ArgumentError 引数エラー
type ArgumentError struct {
	FieldName string
	Message   string
}

// Error Messageを返します
func (ae *ArgumentError) Error() string {
	return ae.Message
}

// ArgError 引数エラーを発生させます
func ArgError(field, message string) *ArgumentError {
	return &ArgumentError{FieldName: field, Message: message}
}

// IsArgError 引数エラーかどうか
func IsArgError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ArgumentError)
	return ok
}
