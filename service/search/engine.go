package search

import (
	"errors"
	"regexp"
	"strings"

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
	Word           optional.String `query:"word"`           // 検索ワード Simple-Query-String-Syntax
	After          optional.Time   `query:"after"`          // 以降(投稿日時) 2020-06-20T00:00:00Z
	Before         optional.Time   `query:"before"`         // 以前(投稿日時)
	In             optional.UUID   `query:"in"`             // 投稿チャンネル
	To             optional.UUID   `query:"to"`             // メンション先
	From           optional.UUID   `query:"from"`           // 投稿者
	Citation       optional.UUID   `query:"citation"`       // 引用しているメッセージ
	Bot            optional.Bool   `query:"bot"`            // 投稿者がBotか
	HasURL         optional.Bool   `query:"hasURL"`         // URLの存在
	HasAttachments optional.Bool   `query:"hasAttachments"` // 添付ファイル
	HasImage       optional.Bool   `query:"hasImage"`       // 添付ファイル（画像）
	HasVideo       optional.Bool   `query:"hasVideo"`       // 添付ファイル（動画）
	HasAudio       optional.Bool   `query:"hasAudio"`       // 添付ファイル（音声ファイル）
	Limit          optional.Int    `query:"limit"`          // 取得件数
	Offset         optional.Int    `query:"offset"`         // 取得Offset
	Sort           optional.String `query:"sort"`           // 並び順 /[-\+]?key/
}

func (q Query) Validate() error {
	return vd.ValidateStruct(&q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(100)),
		// Cannot page through more than 10k hits with From and Size
		// https://www.elastic.co/guide/en/elasticsearch/reference/current/paginate-search-results.html
		vd.Field(&q.Offset, vd.Min(0), vd.Max(9900)),
		vd.Field(&q.Sort, vd.Match(allowedSortKeysRegExp)),
	)
}

// Sort ソート情報
type Sort struct {
	Key  string // 何によってソートするか
	Desc bool   // 降順にするか
}

// GetSortKey ソートに使うキーの情報を抽出します
func (q Query) GetSortKey() Sort {
	if !q.Sort.Valid {
		return Sort{Key: createdAtSortKey, Desc: true}
	}
	match := allowedSortKeysRegExp.FindStringSubmatch(q.Sort.ValueOrZero())
	if match[2] == "" {
		return Sort{Key: createdAtSortKey, Desc: true}
	}
	if match[1] == "-" {
		return Sort{Key: match[2], Desc: !shouldUseDescendingAsDefault(match[2])}
	}
	return Sort{Key: match[2], Desc: shouldUseDescendingAsDefault(match[2])}
}

// Result 検索結果インターフェイス
type Result interface {
	// TotalHits 総ヒット件数
	TotalHits() int64
	// Hits createdAtで降順にソートされた、ヒットしたメッセージ
	Hits() []message.Message
}

const createdAtSortKey = "createdAt" // 作成日時の新しい順
const updatedAtSortKey = "updatedAt" // 更新日時の新しい順

var allowedSortKeys = []string{createdAtSortKey, updatedAtSortKey}
var allowedSortKeysRegExp = regexp.MustCompile("([+-]?)(" + strings.Join(allowedSortKeys, "|") + ")")

// `-`のつかないソートキーを指定した時、対応する値の降順にするか
func shouldUseDescendingAsDefault(key string) bool {
	switch key {
	case createdAtSortKey, updatedAtSortKey:
		return true
	default:
		return false
	}
}
