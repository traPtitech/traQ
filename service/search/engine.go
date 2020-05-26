package search

import (
	"errors"
	"github.com/gofrs/uuid"
)

// ErrServiceUnavailable エラー 現在検索サービスが利用できません
var ErrServiceUnavailable = errors.New("search service is unavailable")

// Engine 検索エンジンインターフェイス
type Engine interface {
	// Do 与えられたクエリで検索を実行します
	Do(q *Query) (Result, error)
	// Available 検索サービスが利用可能かどうかを返します
	Available() bool
	// Close 検索サービスを終了します
	Close() error
}

// Query 検索クエリ TODO
type Query struct {
	// Word 検索ワード (仮置き)
	Word string
}

// Result 検索結果インターフェイス TODO
type Result interface {
	// Get 仮置き
	Get() map[uuid.UUID]string
	// GetMessages() (ms []*model.Message, more bool)
}
