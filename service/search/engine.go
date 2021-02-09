package search

import (
	"errors"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/optional"
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

// Query 検索クエリ
type Query struct {
	// Word 検索ワード (仮置き)
	Word           optional.String `query:"word"`     // 検索ワード Simple-Query-String-Syntax
	After          optional.Time   `query:"after"`    // 以降(投稿日時) 2020-06-20T00:00:00Z
	Before         optional.Time   `query:"before"`   // 以前(投稿日時)
	In             optional.UUID   `query:"in"`       // 投稿チャンネル
	To             optional.UUID   `query:"to"`       // メンション先
	From           optional.UUID   `query:"from"`     // 投稿者
	Citation       optional.UUID   `query:"citation"` // 引用しているメッセージ
	HasURL         optional.Bool   `query:"hasURL"`
	HasAttachments optional.Bool   `query:"hasAttachments"`
	HasImage       optional.Bool   `query:"hasImage"`
	HasVideo       optional.Bool   `query:"hasVideo"`
	HasAudio       optional.Bool   `query:"hasAudio"`
	Limit          optional.Int    `query:"limit"`
	Offset         optional.Int    `query:"offset"`
}

func (q Query) Validate() error {
	return vd.ValidateStruct(&q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(100)),
		// Cannot page through more than 10k hits with From and Size
		// https://www.elastic.co/guide/en/elasticsearch/reference/current/paginate-search-results.html
		vd.Field(&q.Offset, vd.Min(0), vd.Max(9900)),
	)
}

// Result 検索結果インターフェイス
type Result interface {
	// TotalHits 総ヒット件数
	TotalHits() int64
	// Hits createdAtで降順にソートされた、ヒットしたメッセージ
	Hits() []message.Message
}
