package search

import (
	"errors"
	"regexp"
	"strings"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"

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
	Word           optional.Of[string]    `query:"word"`           // 検索ワード Simple-Query-String-Syntax
	After          optional.Of[time.Time] `query:"after"`          // 以降(投稿日時) 2020-06-20T00:00:00Z
	Before         optional.Of[time.Time] `query:"before"`         // 以前(投稿日時)
	In             optional.Of[uuid.UUID] `query:"in"`             // 投稿チャンネル
	To             optional.Of[uuid.UUID] `query:"to"`             // メンション先
	From           optional.Of[uuid.UUID] `query:"from"`           // 投稿者
	Citation       optional.Of[uuid.UUID] `query:"citation"`       // 引用しているメッセージ
	Bot            optional.Of[bool]      `query:"bot"`            // 投稿者がBotか
	HasURL         optional.Of[bool]      `query:"hasURL"`         // URLの存在
	HasAttachments optional.Of[bool]      `query:"hasAttachments"` // 添付ファイル
	HasImage       optional.Of[bool]      `query:"hasImage"`       // 添付ファイル（画像）
	HasVideo       optional.Of[bool]      `query:"hasVideo"`       // 添付ファイル（動画）
	HasAudio       optional.Of[bool]      `query:"hasAudio"`       // 添付ファイル（音声ファイル）
	Limit          optional.Of[int]       `query:"limit"`          // 取得件数
	Offset         optional.Of[int]       `query:"offset"`         // 取得Offset
	Sort           optional.Of[string]    `query:"sort"`           // 並び順 /[-\+]?key/
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

// GetSortKey ソートに使うキーの情報を抽出します
func (q Query) GetSortKey() string {
	if !q.Sort.Valid {
		return createdAtSortKey + ":" + descSortKey
	}
	match := allowedSortKeysRegExp.FindStringSubmatch(q.Sort.ValueOrZero())
	if match[2] == "" {
		return createdAtSortKey + ":" + descSortKey
	}
	if match[1] == "-" {
		if shouldUseDescendingAsDefault(match[2]) {
			return match[2] + ":" + ascSortKey
		}
		return match[2] + ":" + descSortKey
	}
	if shouldUseDescendingAsDefault(match[2]) {
		return match[2] + ":" + descSortKey
	}
	return match[2] + ":" + ascSortKey
}

// Result 検索結果インターフェイス
type Result interface {
	// TotalHits 総ヒット件数
	TotalHits() int64
	// Hits createdAtで降順にソートされた、ヒットしたメッセージ
	Hits() []message.Message
}

const (
	createdAtSortKey = "createdAt" // 作成日時の新しい順
	updatedAtSortKey = "updatedAt" // 更新日時の新しい順
	ascSortKey       = "asc"       // 昇順
	descSortKey      = "desc"      // 降順
)

var (
	allowedSortKeys       = []string{createdAtSortKey, updatedAtSortKey}
	allowedSortKeysRegExp = regexp.MustCompile("([+-]?)(" + strings.Join(allowedSortKeys, "|") + ")")
)

// `-`のつかないソートキーを指定した時、対応する値の降順にするか
func shouldUseDescendingAsDefault(key string) bool {
	switch key {
	case createdAtSortKey, updatedAtSortKey:
		return true
	default:
		return false
	}
}
